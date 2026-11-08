package exercises

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/cleonte/k8slings/k8slings/cluster"
	"github.com/cleonte/k8slings/k8slings/schema"
)

const queryInfo = `[[exercises]]
name='names'
path='exercises/t/names'
topic='t'
mode='query'
hint='h'
timeout='400ms'

[[exercises]]
name='gen'
path='exercises/t/gen'
topic='t'
mode='query'
offline=true
hint='h'
`

var queryFS = fstest.MapFS{
	"exercises/t/names/setup.yaml":         {Data: []byte("kind: Pod\nmetadata: {name: web-1}\n")},
	"exercises/t/names/setup-variant.yaml": {Data: []byte("kind: Pod\nmetadata: {name: extra}\n")},
	"exercises/t/names/check.yaml": {Data: []byte(`
shipped: "not the one"
query:
  command:
    - expr: "verb == 'get'"
      message: "Use kubectl get."
  variant_steps:
    - assert: {resource: pod/extra, expr: "object.status.phase == 'Running'", message: "extra is not Running"}
`)},
	"exercises/t/gen/check.yaml": {Data: []byte(`
shipped: "prints no"
schema: off
query:
  compare: manifest
objects:
  - select: {kind: Deployment, name: web}
    assert:
      - expr: "object.spec.replicas == 3"
        message: "three replicas"
`)},
}

var queryAnswers = fstest.MapFS{
	"solutions/t/names/answer.kubectl": {Data: []byte("kubectl get pods --selector=app=web -o name\n")},
	"solutions/t/gen/answer.kubectl":   {Data: []byte("kubectl create deployment web --image=nginx:1.27 --replicas=3 --dry-run=client -o yaml\n")},
}

func queryEx(t *testing.T, name string) Exercise {
	t.Helper()
	t.Chdir(t.TempDir())
	all, err := ParseInfo([]byte(queryInfo))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range all {
		if e.Name == name {
			return e
		}
	}
	t.Fatal("no such exercise")
	return Exercise{}
}

func queryEnv(k *fakeKube) *Env {
	env := &Env{Embedded: queryFS, Solutions: queryAnswers, Validator: schema.New(queryFS), Poll: time.Millisecond}
	if k != nil {
		env.Kube = k
	}
	return env
}

// variantLive says whether the last of apply and delete in the calls was an apply.
func variantLive(k *fakeKube) bool {
	live := false
	for _, c := range k.calls {
		switch {
		case strings.HasPrefix(c, "apply"):
			live = true
		case strings.HasPrefix(c, "delete"):
			live = false
		}
	}
	return live
}

// names prints the Pods of the state: web-1, and extra while the variant is there.
func names(k *fakeKube) string {
	if variantLive(k) {
		return "pod/extra\npod/web-1\n"
	}
	return "pod/web-1\n"
}

func okKube() *fakeKube {
	k := &fakeKube{nodes: 3, readyAfter: 1}
	k.queryFn = func(args []string, offline bool) (cluster.QueryResult, error) {
		return cluster.QueryResult{Stdout: names(k)}, nil
	}
	return k
}

func writeAnswer(t *testing.T, ex Exercise, text string) {
	t.Helper()
	if err := os.MkdirAll("exercises/t/"+ex.Name, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ex.File(), []byte("# I AM NOT DONE\n"+text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestQueryFilesLiveWhereTheyShould(t *testing.T) {
	ex := queryEx(t, "names")
	if ex.File() != "exercises/t/names/answer.kubectl" && !strings.HasSuffix(ex.File(), "answer.kubectl") {
		t.Errorf("File = %s", ex.File())
	}
	if ex.SolutionPath() != "solutions/t/names/answer.kubectl" {
		t.Errorf("SolutionPath = %s", ex.SolutionPath())
	}
	for _, f := range []string{"check.yaml", "setup.yaml", "setup-variant.yaml"} {
		if !EmbeddedOnly("exercises/t/names/" + f) {
			t.Errorf("%s must stay in the binary", f)
		}
	}
	if EmbeddedOnly("exercises/t/names/answer.kubectl") {
		t.Error("answer.kubectl is the file the learner edits")
	}
}

func TestQuerySeedsOnTheFirstRunThenComparesAndPasses(t *testing.T) {
	ex := queryEx(t, "names")
	k := okKube()
	writeAnswer(t, ex, "kubectl get pods -l app=web -o name\n")
	if _, err := ex.Run(context.Background(), queryEnv(k)); err != nil {
		t.Fatal(err)
	}
	if k.seedCalls != 1 {
		t.Errorf("seed calls = %d, want 1", k.seedCalls)
	}
	if !passed("names") {
		t.Error("a pass must be recorded")
	}
	// The learner's command runs in the exercise's own namespace.
	var learner string
	for _, c := range k.calls {
		if strings.HasPrefix(c, "query get pods") {
			learner = c
			break
		}
	}
	if !strings.HasSuffix(learner, "--namespace=k8slings-names") {
		t.Errorf("the namespace is added by k8slings: %q", learner)
	}
}

func TestQueryDoesNotSeedAgainOnceSeeded(t *testing.T) {
	ex := queryEx(t, "names")
	k := okKube()
	k.seeded = true
	if err := recordSeed("names", nil); err != nil {
		t.Fatal(err)
	}
	writeAnswer(t, ex, "kubectl get pods -l app=web -o name\n")
	if _, err := ex.Run(context.Background(), queryEnv(k)); err != nil {
		t.Fatal(err)
	}
	if k.seedCalls != 0 {
		t.Errorf("seeded again: %d", k.seedCalls)
	}
}

func TestQueryCheckAllNeverSeeds(t *testing.T) {
	ex := queryEx(t, "names")
	k := okKube()
	writeAnswer(t, ex, "kubectl get pods -l app=web -o name\n")
	_, err := ex.Run(NoSeed(context.Background()), queryEnv(k))
	if !IsSkipped(err) || k.seedCalls != 0 {
		t.Fatalf("err=%v seeds=%d: an exercise that was not started is skipped", err, k.seedCalls)
	}
	if passed("names") {
		t.Error("nothing passed")
	}
}

func TestQueryWithoutACluster(t *testing.T) {
	ex := queryEx(t, "names")
	writeAnswer(t, ex, "kubectl get pods\n")
	_, err := ex.Run(context.Background(), queryEnv(nil))
	if !IsSkipped(err) || !strings.Contains(err.Error(), "cluster up") {
		t.Fatalf("err = %v", err)
	}
}

func TestQueryRefusalsNeverReachKubectl(t *testing.T) {
	ex := queryEx(t, "names")
	for text, want := range map[string]string{
		"kubectl delete pod web-1\n":               "changes the cluster",
		"kubectl get pods | grep web\n":            "not a shell",
		"kubectl get pods -n kube-system\n":        "another namespace",
		"kubectl get pods --kubeconfig=/x\n":       "not available",
		"kubectl get pods -o name -- --server=x\n": "--",
		"# nothing yet\n":                          "empty",
	} {
		k := okKube()
		writeAnswer(t, ex, text)
		_, err := ex.Run(context.Background(), queryEnv(k))
		var failed *FailedError
		if !errors.As(err, &failed) || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: err = %v, want a failure with %q", text, err, want)
		}
		for _, c := range k.calls {
			if strings.HasPrefix(c, "query") {
				t.Errorf("%q reached kubectl: %s", text, c)
			}
		}
	}
}

func TestQueryShowsBothOutputsWhenTheyDiffer(t *testing.T) {
	ex := queryEx(t, "names")
	k := okKube()
	k.queryFn = func(args []string, _ bool) (cluster.QueryResult, error) {
		if strings.Contains(strings.Join(args, " "), "--selector=app=web") { // the reference
			return cluster.QueryResult{Stdout: "pod/web-1\n"}, nil
		}
		return cluster.QueryResult{Stdout: "pod/web-1\npod/db-1\n", Stderr: "warning: x\n"}, nil
	}
	writeAnswer(t, ex, "kubectl get pods -o name\n")
	_, err := ex.Run(context.Background(), queryEnv(k))
	var failed *FailedError
	if !errors.As(err, &failed) {
		t.Fatalf("err = %v", err)
	}
	all := strings.Join(failed.Details, "\n")
	for _, want := range []string{"Your command printed:", "  pod/db-1", "kubectl said:", "warning: x", "Expected:", "  pod/web-1"} {
		if !strings.Contains(all, want) {
			t.Errorf("details lack %q:\n%s", want, all)
		}
	}
	if passed("names") {
		t.Error("a failure must not be recorded as a pass")
	}
}

func TestQueryShowsKubectlsOwnError(t *testing.T) {
	ex := queryEx(t, "names")
	k := okKube()
	k.queryFn = func([]string, bool) (cluster.QueryResult, error) {
		return cluster.QueryResult{Stderr: "error: the server doesn't have a resource type \"podz\"\nmore\n"}, errors.New("exit status 1")
	}
	writeAnswer(t, ex, "kubectl get podz\n")
	_, err := ex.Run(context.Background(), queryEnv(k))
	if err == nil || !strings.Contains(err.Error(), `kubectl failed: error: the server doesn't have a resource type "podz"`) {
		t.Fatalf("err = %v", err)
	}
}

func TestQueryLimits(t *testing.T) {
	for name, res := range map[string]cluster.QueryResult{
		"took more than":      {TimedOut: true},
		"printed more than 1": {Truncated: true, Stdout: "x"},
	} {
		ex := queryEx(t, "names")
		k := okKube()
		k.queryFn = func([]string, bool) (cluster.QueryResult, error) {
			if res.TimedOut {
				return res, context.DeadlineExceeded
			}
			return res, nil
		}
		writeAnswer(t, ex, "kubectl get pods\n")
		_, err := ex.Run(context.Background(), queryEnv(k))
		if err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestQueryWithoutKubectlIsSkipped(t *testing.T) {
	ex := queryEx(t, "names")
	k := okKube()
	k.queryFn = func([]string, bool) (cluster.QueryResult, error) { return cluster.QueryResult{}, cluster.ErrNoKubectl }
	writeAnswer(t, ex, "kubectl get pods\n")
	if _, err := ex.Run(context.Background(), queryEnv(k)); !IsSkipped(err) || !strings.Contains(err.Error(), "kubectl") {
		t.Fatalf("err = %v", err)
	}
}

func TestAHardCodedAnswerFailsTheVariant(t *testing.T) {
	ex := queryEx(t, "names")
	k := okKube()
	k.queryFn = func(args []string, _ bool) (cluster.QueryResult, error) {
		if strings.Contains(strings.Join(args, " "), "--selector=app=web") { // the reference: follows the state
			return cluster.QueryResult{Stdout: names(k)}, nil
		}
		return cluster.QueryResult{Stdout: "pod/web-1\n"}, nil // the learner typed today's name
	}
	writeAnswer(t, ex, "kubectl get pods -o name\n")
	_, err := ex.Run(context.Background(), queryEnv(k))
	var failed *FailedError
	if !errors.As(err, &failed) || !strings.Contains(err.Error(), "other objects are in the namespace") {
		t.Fatalf("err = %v", err)
	}
	if passed("names") {
		t.Error("a hard-coded answer must not be recorded as a pass")
	}
}

func TestVariantWaitsForItsStepsAndAlwaysTakesTheObjectsAway(t *testing.T) {
	ex := queryEx(t, "names")
	k := okKube()
	k.readyAfter = 4 // pod/extra reports Running from the fourth read on
	var readsAtVariantQuery int32
	k.queryFn = func(args []string, _ bool) (cluster.QueryResult, error) {
		if variantLive(k) {
			atomic.StoreInt32(&readsAtVariantQuery, k.reads.Load())
		}
		return cluster.QueryResult{Stdout: names(k)}, nil
	}
	writeAnswer(t, ex, "kubectl get pods -l app=web -o name\n")
	if _, err := ex.Run(context.Background(), queryEnv(k)); err != nil {
		t.Fatal(err)
	}
	if readsAtVariantQuery < 4 {
		t.Errorf("the variant was compared after %d reads: its steps were not waited for", readsAtVariantQuery)
	}
	if variantLive(k) {
		t.Error("the variant objects must be taken away again")
	}
	if len(k.deleted) < 2 || !strings.Contains(string(k.deleted[len(k.deleted)-1]), "name: extra") {
		t.Errorf("deleted = %q", k.deleted)
	}
}

func TestVariantIsTakenAwayAlsoWhenTheComparisonFails(t *testing.T) {
	ex := queryEx(t, "names")
	k := okKube()
	k.queryFn = func(args []string, _ bool) (cluster.QueryResult, error) {
		if strings.Contains(strings.Join(args, " "), "--selector=app=web") {
			return cluster.QueryResult{Stdout: names(k)}, nil
		}
		return cluster.QueryResult{Stdout: "pod/web-1\n"}, nil
	}
	writeAnswer(t, ex, "kubectl get pods -o name\n")
	_, _ = ex.Run(context.Background(), queryEnv(k))
	if variantLive(k) {
		t.Error("the variant is still there after a failed run")
	}
}

func TestVariantIsTakenAwayWhenTheRunIsCancelled(t *testing.T) {
	ex := queryEx(t, "names")
	k := okKube()
	ctx, cancel := context.WithCancel(context.Background())
	k.queryFn = func(args []string, _ bool) (cluster.QueryResult, error) {
		if variantLive(k) {
			cancel() // a save in `watch` while the variant is there
			return cluster.QueryResult{}, context.Canceled
		}
		return cluster.QueryResult{Stdout: names(k)}, nil
	}
	writeAnswer(t, ex, "kubectl get pods -l app=web -o name\n")
	_, _ = ex.Run(ctx, queryEnv(k))
	if variantLive(k) {
		t.Error("a cancelled run left the variant in the namespace")
	}
}

func TestOfflineQueryChecksTheOutputAsAManifest(t *testing.T) {
	ex := queryEx(t, "gen")
	k := &fakeKube{nodes: 3}
	var offline bool
	k.queryFn = func(args []string, off bool) (cluster.QueryResult, error) {
		offline = off
		return cluster.QueryResult{Stdout: "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: web\nspec:\n  replicas: 3\n  selector: {matchLabels: {app: web}}\n  template:\n    metadata: {labels: {app: web}}\n    spec:\n      containers: [{name: nginx, image: 'nginx:1.27'}]\n"}, nil
	}
	writeAnswer(t, ex, "kubectl create deployment web --image=nginx:1.27 --replicas=3 --dry-run=client -o yaml\n")
	if _, err := ex.Run(context.Background(), queryEnv(k)); err != nil {
		t.Fatal(err)
	}
	if !offline || k.seedCalls != 0 {
		t.Errorf("offline=%v seeds=%d: no cluster, no seeding", offline, k.seedCalls)
	}
	// Whether the cluster is up does not matter, only that kubectl can run.
	k.verifyErr = errors.New("no cluster")
	if _, err := ex.Run(context.Background(), queryEnv(k)); err != nil {
		t.Fatalf("an offline exercise needs no cluster: %v", err)
	}
	if _, err := ex.Run(context.Background(), queryEnv(nil)); !IsSkipped(err) {
		t.Fatalf("without any way to run kubectl it is skipped: %v", err)
	}
}

func TestOfflineQueryRefusesWhatNeedsACluster(t *testing.T) {
	ex := queryEx(t, "gen")
	writeAnswer(t, ex, "kubectl get pods\n")
	_, err := ex.Run(context.Background(), queryEnv(&fakeKube{nodes: 3}))
	if err == nil || !strings.Contains(err.Error(), "no cluster") {
		t.Fatalf("err = %v", err)
	}
}

func TestOfflineManifestMistakesAreFailures(t *testing.T) {
	ex := queryEx(t, "gen")
	for out, want := range map[string]string{
		"": "no Kubernetes object",
		"apiVersion: apps/v1\nkind: Deployment\nmetadata: {name: web}\nspec: {replicas: 1}\n": "three replicas",
		"a: [unclosed": "not YAML",
	} {
		k := &fakeKube{nodes: 3}
		k.queryFn = func([]string, bool) (cluster.QueryResult, error) { return cluster.QueryResult{Stdout: out}, nil }
		writeAnswer(t, ex, "kubectl create deployment web --image=nginx:1.27 --dry-run=client\n")
		_, err := ex.Run(context.Background(), queryEnv(k))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: err = %v, want %q", out, err, want)
		}
	}
}

func TestNormalize(t *testing.T) {
	for _, tc := range []struct {
		in   string
		sort bool
		want string
	}{
		{"a  \nb\t\n\n\n", false, "a\nb"},
		{"a\r\nb\r\n", false, "a\nb"},
		{"b\na\n", false, "b\na"},
		{"b\na\n", true, "a\nb"},
		{"", false, ""},
		{"  indented\n", false, "  indented"},
	} {
		if got := normalize(tc.in, tc.sort); got != tc.want {
			t.Errorf("normalize(%q, %v) = %q, want %q", tc.in, tc.sort, got, tc.want)
		}
	}
}

func TestSideBySideShowsOnlyTheFirstLines(t *testing.T) {
	long := strings.Repeat("line\n", 50)
	out := strings.Join(sideBySide(long, "x", ""), "\n")
	if !strings.Contains(out, "... 31 more lines") {
		t.Errorf("a long output is cut:\n%s", out)
	}
	if !strings.Contains(strings.Join(sideBySide("", "x", ""), "\n"), "(nothing)") {
		t.Error("an empty output says so")
	}
}

func TestParseInfoQueryRules(t *testing.T) {
	if _, err := ParseInfo([]byte("[[exercises]]\nname='a'\npath='p'\ntopic='t'\nmode='manifest'\noffline=true\nhint='h'\n")); err == nil {
		t.Error("only a query exercise can be offline")
	}
}

func TestTheRulesOfTheExerciseAreReportedWithTheOutput(t *testing.T) {
	ex := queryEx(t, "names")
	k := okKube()
	k.queryFn = func(args []string, _ bool) (cluster.QueryResult, error) {
		if strings.Contains(strings.Join(args, " "), "--selector=app=web") {
			return cluster.QueryResult{Stdout: "pod/web-1\n"}, nil
		}
		return cluster.QueryResult{Stdout: "pod/other\n"}, nil
	}
	// A command that breaks a rule and prints the wrong thing: both are said.
	writeAnswer(t, ex, "kubectl describe pod web-1\n")
	_, err := ex.Run(context.Background(), queryEnv(k))
	var failed *FailedError
	if !errors.As(err, &failed) || len(failed.Lines) != 2 || !strings.Contains(failed.Lines[0], "output is not") || failed.Lines[1] != "Use kubectl get." {
		t.Fatalf("err = %v", err)
	}
}

func TestACommandThatBreaksARuleFailsEvenWhenItsOutputIsRight(t *testing.T) {
	ex := queryEx(t, "names")
	k := okKube()
	k.queryFn = func(args []string, _ bool) (cluster.QueryResult, error) {
		return cluster.QueryResult{Stdout: "pod/web-1\n"}, nil // the same, whatever the command
	}
	writeAnswer(t, ex, "kubectl logs web-1\n") // verb is not get
	_, err := ex.Run(context.Background(), queryEnv(k))
	if err == nil || !strings.Contains(err.Error(), "Use kubectl get.") {
		t.Fatalf("err = %v", err)
	}
	if passed("names") {
		t.Error("a rule broken is not a pass")
	}
}
