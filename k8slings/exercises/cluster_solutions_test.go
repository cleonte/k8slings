//go:build cluster

package exercises_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cleonte/k8slings"
	"github.com/cleonte/k8slings/k8slings/check"
	"github.com/cleonte/k8slings/k8slings/cluster"
	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/cleonte/k8slings/k8slings/schema"
)

// One cluster serves every proof in this package, so that CI pays for creating
// it once. It is deleted when the tests are over, unless K8SLINGS_KEEP is set.
var proof struct {
	once sync.Once
	cl   *cluster.Cluster
	err  error
	log  string
}

func proofCluster(t *testing.T, ctx context.Context) *cluster.Cluster {
	t.Helper()
	proof.once.Do(func() {
		dir, err := os.MkdirTemp("", "k8slings-proof-")
		if err != nil {
			proof.err = err
			return
		}
		proof.cl = &cluster.Cluster{Name: "k8slings-proof", Kubeconfig: filepath.Join(dir, "kubeconfig"), Runner: cluster.ExecRunner{}}
		var up bytes.Buffer
		if err := proof.cl.Up(ctx, cluster.UpOptions{Out: &up}); err != nil {
			proof.err = fmt.Errorf("cluster up: %w\n%s", err, up.String())
		}
		proof.log = up.String()
	})
	if proof.err != nil {
		t.Fatal(proof.err)
	}
	return proof.cl
}

func TestMain(m *testing.M) {
	code := m.Run()
	if proof.cl != nil && os.Getenv("K8SLINGS_KEEP") == "" {
		_ = proof.cl.Down(context.Background())
	}
	os.Exit(code)
}

// TestClusterSolutions proves the cluster exercises on a real cluster: as
// shipped each one must fail (within a short timeout, and by the exercise's own
// check, not by a broken cluster), and with its solution each one must pass.
// It is run by `make cluster-test`, and needs Docker, kind and kubectl:
//
//	go test -tags cluster -run TestClusterSolutions -timeout 30m ./k8slings/exercises/
//
// K8SLINGS_KEEP=1 leaves the cluster (k8slings-proof) up afterwards, and a later
// run reuses it. The fix proofs share it.
func TestClusterSolutions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()

	cl := proofCluster(t, ctx)

	validator := schema.New(k8slings.Schemas)
	solve := &exercises.Env{Embedded: k8slings.FS, Validator: validator, Kube: cl}
	broken := &exercises.Env{Embedded: k8slings.FS, Validator: validator, Kube: cl, Timeout: 15 * time.Second}

	for _, ex := range load(t) {
		if ex.Mode != exercises.ModeCluster {
			continue
		}
		t.Run(ex.Name, func(t *testing.T) {
			src, err := fs.ReadFile(k8slings.FS, ex.Source())
			if err != nil {
				t.Fatal(err)
			}
			_, err = ex.RunSource(ctx, broken, src)
			var failed *exercises.FailedError
			if !errors.As(err, &failed) {
				t.Fatalf("the unsolved exercise must fail with a FailedError, got %v", err)
			}
			t.Logf("fails as intended: %s", failed.Lines[0])
			// It must fail for the reason it teaches, not just fail: a slow cluster
			// fails every exercise. check.yaml says what the first problem is.
			raw, err := fs.ReadFile(k8slings.FS, ex.CheckPath())
			if err != nil {
				t.Fatal(err)
			}
			compiled, err := check.Load(raw)
			if err != nil {
				t.Fatal(err)
			}
			if compiled.Shipped == "" {
				t.Fatalf("%s needs a `shipped:` line in its check.yaml: what the first problem of the unsolved exercise says", ex.Name)
			}
			if !strings.Contains(failed.Lines[0], compiled.Shipped) {
				t.Fatalf("the unsolved exercise fails, but not for the reason it teaches:\n  got:  %s\n  want: ...%s...", failed.Lines[0], compiled.Shipped)
			}

			sol, err := fs.ReadFile(k8slings.Solutions, ex.SolutionPath())
			if err != nil {
				t.Fatal(err)
			}
			start := time.Now()
			if _, err := ex.RunSource(ctx, solve, sol); err != nil {
				// Show what the cluster looked like, not only which step failed.
				var f *exercises.FailedError
				if errors.As(err, &f) {
					t.Fatalf("the solution must pass: %v\n%s", err, strings.Join(f.Details, "\n"))
				}
				t.Fatalf("the solution must pass: %v", err)
			}
			t.Logf("solution passes in %s", time.Since(start).Round(time.Second))
		})
	}
}

// TestFixSolutions proves the fix exercises on a real cluster: the first run
// seeds the broken state, the check then fails for the reason `shipped` names,
// the solution script repairs it and the check passes, and each shortcut script
// (a way to cheat) is refused with the message it declares.
//
//	go test -tags cluster -run TestFixSolutions -timeout 30m ./k8slings/exercises/
func TestFixSolutions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()

	cl := proofCluster(t, ctx)
	env := &exercises.Env{Embedded: k8slings.FS, Validator: schema.New(k8slings.Schemas), Kube: cl}

	// until polls: a cluster takes a few seconds to react to anything.
	until := func(t *testing.T, limit time.Duration, what string, f func() (done bool, why string)) {
		t.Helper()
		deadline := time.Now().Add(limit)
		var why string
		for {
			var done bool
			if done, why = f(); done {
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("timed out after %s waiting for %s; last: %s", limit, what, why)
			}
			time.Sleep(2 * time.Second)
		}
	}
	firstProblem := func(err error) string {
		var failed *exercises.FailedError
		if errors.As(err, &failed) && len(failed.Lines) > 0 {
			return failed.Lines[0]
		}
		if err != nil {
			return err.Error()
		}
		return ""
	}

	for _, ex := range load(t) {
		if ex.Mode != exercises.ModeFix {
			continue
		}
		t.Run(ex.Name, func(t *testing.T) {
			t.Chdir(t.TempDir()) // the progress file lives in the working directory
			raw, _ := fs.ReadFile(k8slings.FS, ex.CheckPath())
			compiled, err := check.Load(raw)
			if err != nil {
				t.Fatal(err)
			}

			// Start clean: a cluster kept from an earlier run still has the namespace
			// labelled as seeded, and this run has no progress file.
			if out, err := cl.Kubectl(ctx, "delete", "namespace", ex.Namespace(), "--ignore-not-found", "--wait=true"); err != nil {
				t.Fatalf("cannot clear the namespace: %s", out)
			}

			// 1. The first run breaks the cluster, and says so.
			if _, err := ex.Run(ctx, env); !exercises.IsSeeded(err) {
				t.Fatalf("the first run must seed the exercise, got %v", err)
			}
			if ex.State() != exercises.Pending {
				t.Fatal("a seeded exercise is not done")
			}

			// 2. It fails for the reason it teaches (it takes a moment for the
			// cluster to show it), and never seeds again by itself.
			var shipped string
			until(t, time.Minute, "the shipped problem", func() (bool, string) {
				_, err := ex.Run(ctx, env)
				shipped = firstProblem(err)
				return strings.Contains(shipped, compiled.Shipped), shipped
			})
			t.Logf("fails as intended: %s", shipped)
			// It must keep failing: a broken exercise that passes now and then
			// (a crash-looping container is briefly Ready) is a false repair.
			for i := range 12 {
				if _, err := ex.Run(ctx, env); err == nil {
					t.Fatalf("the unrepaired exercise PASSED on look %d: a repair must hold", i+1)
				}
				time.Sleep(time.Second) // spread the looks out, so that one can land on a brief Ready
			}

			// 3. The solution repairs it.
			script, err := fs.ReadFile(k8slings.Solutions, ex.SolutionPath())
			if err != nil {
				t.Fatal(err)
			}
			if out, err := cl.RunScript(ctx, ex.Namespace(), script); err != nil {
				t.Fatalf("the solution script failed: %v\n%s", err, out)
			}
			start := time.Now()
			until(t, 2*time.Minute, "the solution to pass", func() (bool, string) {
				_, err := ex.Run(ctx, env)
				return err == nil, firstProblem(err)
			})
			t.Logf("solution passes in %s", time.Since(start).Round(time.Second))
			if ex.State() != exercises.Done {
				t.Fatal("a repaired exercise must be recorded as done")
			}

			// 4. Each shortcut is refused, with the message it declares.
			shortcuts, _ := fs.Glob(k8slings.Solutions, path.Join(path.Dir(ex.SolutionPath()), "shortcut-*.sh"))
			for _, f := range shortcuts {
				t.Run(path.Base(f), func(t *testing.T) {
					data, _ := fs.ReadFile(k8slings.Solutions, f)
					first, _, _ := strings.Cut(string(data), "\n")
					want := rejectedRe.FindStringSubmatch(first)[1]

					// A shortcut that changes something outside the namespace (a node
					// label) leaves a cleanup.sh next to it, run when the shortcut is done.
					if cleanup, err := fs.ReadFile(k8slings.Solutions, path.Join(path.Dir(f), "cleanup.sh")); err == nil {
						t.Cleanup(func() {
							if out, err := cl.RunScript(ctx, ex.Namespace(), cleanup); err != nil {
								t.Errorf("cleanup.sh failed: %v\n%s", err, out)
							}
						})
					}
					if err := ex.Reseed(ctx, env); err != nil {
						t.Fatal(err)
					}
					if out, err := cl.RunScript(ctx, ex.Namespace(), data); err != nil {
						t.Fatalf("the shortcut script failed: %v\n%s", err, out)
					}
					passed := false
					until(t, 45*time.Second, `the refusal "`+want+`"`, func() (bool, string) {
						_, err := ex.Run(ctx, env)
						if err == nil {
							passed = true
							return true, "the shortcut PASSED"
						}
						return strings.Contains(firstProblem(err), want), firstProblem(err)
					})
					if passed {
						t.Fatal("the shortcut was accepted as a repair")
					}
					if ex.State() != exercises.Pending {
						t.Fatal("a shortcut must not be recorded as done")
					}
				})
			}
		})
	}
}
