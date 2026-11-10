//go:build cluster

package exercises_test

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/cleonte/k8slings"
	"github.com/cleonte/k8slings/k8slings/check"
	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/cleonte/k8slings/k8slings/query"
	"github.com/cleonte/k8slings/k8slings/schema"
)

// recordedPass says whether the progress file of the working directory has a pass
// for the exercise.
func recordedPass(name string) bool {
	data, err := os.ReadFile(exercises.ProgressFile)
	if err != nil {
		return false
	}
	var p struct{ Done map[string]string }
	_ = json.Unmarshal(data, &p)
	_, ok := p.Done[name]
	return ok
}

// stateOf lists the objects of a namespace with their UIDs, which is what a
// variant must give back exactly.
func stateOf(t *testing.T, ctx context.Context, ns string, kubectl func(context.Context, ...string) (string, error)) string {
	t.Helper()
	out, err := kubectl(ctx, "get", "all,configmap,secret,serviceaccount,role,rolebinding,networkpolicy,persistentvolumeclaim,poddisruptionbudget,cronjob,job", "-n", ns,
		"-o", `jsonpath={range .items[*]}{.kind}/{.metadata.name}={.metadata.uid}{"\n"}{end}`)
	if err != nil {
		t.Fatalf("cannot list the state of %s: %s", ns, out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	slices.Sort(lines)
	return strings.Join(lines, "\n")
}

// TestQuerySolutions proves the query exercises on a real cluster: the shipped
// answer fails for the reason `shipped` names, the reference answer passes (and
// prints the same thing twice), the state is exactly the same after the variant
// has come and gone, and every shortcut (a hard-coded answer, a command that
// cheats) is refused with the message it declares.
//
//	go test -tags cluster -run TestQuerySolutions -timeout 60m ./k8slings/exercises/
func TestQuerySolutions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Minute)
	defer cancel()

	cl := proofCluster(t, ctx)
	solve := &exercises.Env{Embedded: k8slings.FS, Solutions: k8slings.Solutions, Validator: schema.New(k8slings.Schemas), Kube: cl}

	for _, ex := range load(t) {
		if ex.Mode != exercises.ModeQuery || !inShard(t, ex.Name) || !coveredHere(t, ctx, cl, ex) {
			continue
		}
		t.Run(ex.Name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			raw, _ := fs.ReadFile(k8slings.FS, ex.CheckPath())
			compiled, err := check.Load(raw)
			if err != nil {
				t.Fatal(err)
			}
			write := func(content []byte) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(ex.File()), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(ex.File(), content, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			problems := func(err error) string {
				var failed *exercises.FailedError
				switch {
				case errors.As(err, &failed):
					return strings.Join(append(append([]string{}, failed.Lines...), failed.Details...), "\n")
				case err != nil:
					return err.Error()
				}
				return ""
			}

			if !ex.Offline {
				if err := cl.ReleaseFinalizers(ctx, ex.Namespace()); err != nil {
					t.Fatalf("cannot release what holds the namespace: %v", err)
				}
				if out, err := cl.Kubectl(ctx, "delete", "namespace", ex.Namespace(), "--ignore-not-found", "--wait=true", "--timeout=3m"); err != nil {
					t.Fatalf("cannot clear the namespace: %s", out)
				}
			}

			// 1. As shipped it fails, for the reason it teaches. The first run also
			// seeds the state.
			shippedSrc, err := fs.ReadFile(k8slings.FS, ex.Source())
			if err != nil {
				t.Fatal(err)
			}
			write(shippedSrc)
			_, err = ex.Run(ctx, solve)
			if err == nil || exercises.IsSkipped(err) || !strings.Contains(problems(err), compiled.Shipped) {
				t.Fatalf("the shipped answer must fail with %q, got: %v", compiled.Shipped, problems(err))
			}
			t.Logf("fails as intended: %s", strings.SplitN(problems(err), "\n", 2)[0])

			// 2. The reference prints the same thing every time. A volatile output
			// (ages, timestamps) would make every comparison a coin toss.
			ref, err := fs.ReadFile(k8slings.Solutions, ex.SolutionPath())
			if err != nil {
				t.Fatal(err)
			}
			opt := query.Options{AllNamespaces: compiled.Query.AllNamespaces}
			if !ex.Offline {
				opt.Namespace = ex.Namespace()
			}
			cmd, err := query.Parse(string(ref), opt)
			if err != nil {
				t.Fatalf("the reference answer is refused: %v", err)
			}
			var outputs []string
			for range 3 {
				res, err := cl.Query(ctx, cmd.Args, ex.Offline || cmd.Offline)
				if err != nil {
					t.Fatalf("the reference answer fails: %v\n%s", err, res.Stderr)
				}
				outputs = append(outputs, res.Stdout)
				time.Sleep(2 * time.Second)
			}
			for _, o := range outputs[1:] {
				if o != outputs[0] {
					t.Fatalf("the reference answer prints something different each time:\n%s\n---\n%s", outputs[0], o)
				}
			}
			if strings.TrimSpace(outputs[0]) == "" {
				t.Fatal("the reference answer prints nothing: an empty output proves nothing")
			}

			// 3. The reference passes, and the variant leaves no trace.
			write(ref)
			var before string
			if !ex.Offline {
				before = stateOf(t, ctx, ex.Namespace(), cl.Kubectl)
			}
			start := time.Now()
			if _, err := ex.Run(ctx, solve); err != nil {
				t.Fatalf("the reference answer must pass: %s", problems(err))
			}
			t.Logf("solution passes in %s", time.Since(start).Round(time.Second))
			if !recordedPass(ex.Name) {
				t.Fatal("a passing answer must be recorded")
			}
			if !ex.Offline {
				if after := stateOf(t, ctx, ex.Namespace(), cl.Kubectl); after != before {
					t.Fatalf("the variant left the namespace different:\nbefore:\n%s\nafter:\n%s", before, after)
				}
			}

			// 4. Every shortcut is refused, with the message it declares.
			shortcuts, _ := fs.Glob(k8slings.Solutions, path.Join(path.Dir(ex.SolutionPath()), "shortcut-*.kubectl"))
			for _, f := range shortcuts {
				t.Run(path.Base(f), func(t *testing.T) {
					data, _ := fs.ReadFile(k8slings.Solutions, f)
					first, _, _ := strings.Cut(string(data), "\n")
					want := rejectedRe.FindStringSubmatch(first)[1]
					if err := os.Remove(exercises.ProgressFile); err != nil && !os.IsNotExist(err) {
						t.Fatal(err)
					}
					if !ex.Offline {
						if err := ex.Reseed(ctx, solve); err != nil {
							t.Fatal(err)
						}
					}
					write(data)
					_, err := ex.Run(ctx, solve)
					if err == nil {
						t.Fatal("the shortcut was accepted")
					}
					if !strings.Contains(problems(err), want) {
						t.Fatalf("the shortcut must be refused with %q, got: %s", want, problems(err))
					}
					if recordedPass(ex.Name) {
						t.Fatal("a refused shortcut must not be recorded as a pass")
					}
					t.Logf("refused: %s", want)
				})
			}
		})
	}
}
