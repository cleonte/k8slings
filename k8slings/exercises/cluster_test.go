package exercises

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/cleonte/k8slings/k8slings/cluster"
	"github.com/cleonte/k8slings/k8slings/schema"
)

// fakeKube is a cluster that records what the runner does to it.
type fakeKube struct {
	verifyErr  error
	nodes      int
	applyErr   bool
	applyOut   string
	calls      []string
	reads      atomic.Int32
	readyAfter int32 // the pod reports Running from this read on

	// fix exercises
	seeded    bool
	seedCalls int
	seedSetup string // the steps given to Seed, joined by "|"
	seedErr   error  // Seed fails with this, and leaves the cluster unseeded
	seedBlock bool   // Seed waits until its context is cancelled
	liveUID   string // what the cluster says the UID of an object is
	flap      bool   // the pod is Running on odd reads and Pending on even ones

	// query exercises
	queryFn func(args []string, offline bool) (cluster.QueryResult, error)
	deleted [][]byte // the manifests given to Delete

	addons map[string]bool // the add-ons that are installed and ready
}

func (f *fakeKube) ReadyAddons(context.Context) map[string]bool { return f.addons }

func (f *fakeKube) Delete(_ context.Context, ns string, manifest []byte) error {
	f.calls = append(f.calls, "delete "+ns)
	f.deleted = append(f.deleted, manifest)
	return nil
}

func (f *fakeKube) Query(_ context.Context, args []string, offline bool) (cluster.QueryResult, error) {
	f.calls = append(f.calls, "query "+strings.Join(args, " "))
	if f.queryFn == nil {
		return cluster.QueryResult{}, nil
	}
	return f.queryFn(args, offline)
}

func (f *fakeKube) Seed(ctx context.Context, ns, name string, steps [][]byte) error {
	f.calls = append(f.calls, "seed "+ns+" "+name)
	f.seedCalls++
	var parts []string
	for _, s := range steps {
		parts = append(parts, string(s))
	}
	f.seedSetup = strings.Join(parts, "|")
	f.seeded = true // the label is put on first, so it stays when the seeding fails
	if f.seedBlock {
		<-ctx.Done()
		return ctx.Err()
	}
	return f.seedErr
}
func (f *fakeKube) Seeded(context.Context, string, string) bool { return f.seeded }
func (f *fakeKube) UID(_ context.Context, _, resource string) (string, error) {
	return "uid-" + resource, nil
}

func (f *fakeKube) Verify(context.Context) error { return f.verifyErr }
func (f *fakeKube) Nodes(context.Context) (int, error) {
	return f.nodes, nil
}
func (f *fakeKube) ResetNamespace(_ context.Context, ns string) error {
	f.calls = append(f.calls, "reset "+ns)
	return nil
}
func (f *fakeKube) Apply(_ context.Context, ns string, _ []byte) (string, error) {
	f.calls = append(f.calls, "apply "+ns)
	if f.applyErr {
		return f.applyOut, errors.New("exit status 1")
	}
	return "pod/web created", nil
}
func (f *fakeKube) Diagnose(context.Context, string) []string {
	return []string{"Pods:", "  web  Pending  ready 0/1"}
}
func (f *fakeKube) Kubectl(_ context.Context, args ...string) (string, error) {
	f.calls = append(f.calls, strings.Join(args, " "))
	if strings.Contains(strings.Join(args, " "), "jsonpath={.metadata.uid}") {
		if f.liveUID != "" {
			return f.liveUID, nil
		}
		return "uid-pod/web", nil
	}
	n := f.reads.Add(1)
	if f.flap {
		if n%2 == 1 {
			return `{"status":{"phase":"Running"}}`, nil
		}
		return `{"status":{"phase":"Pending"}}`, nil
	}
	if n >= f.readyAfter {
		return `{"status":{"phase":"Running"}}`, nil
	}
	return `{"status":{"phase":"Pending"}}`, nil
}

const clusterInfo = `[[exercises]]
name='web'
path='exercises/t/web'
topic='t'
mode='cluster'
hint='h'
timeout='400ms'
`

var clusterFS = fstest.MapFS{
	"exercises/t/web/check.yaml": {Data: []byte(`
schema: off
objects:
  - select: {kind: Pod, name: web}
    assert:
      - expr: object.spec.containers[0].image == 'nginx:1.27'
        message: "use nginx 1.27"
steps:
  - assert: {resource: pod/web, expr: "object.status.phase == 'Running'", message: "the Pod is not Running yet"}
`)},
}

const goodPod = "apiVersion: v1\nkind: Pod\nmetadata: {name: web}\nspec: {containers: [{name: c, image: 'nginx:1.27'}]}\n"
const typoPod = "apiVersion: v1\nkind: Pod\nmetadata: {name: web}\nspec: {containers: [{name: c, image: 'nginx:1.2'}]}\n"

func clusterExercise(t *testing.T) Exercise {
	t.Helper()
	all, err := ParseInfo([]byte(clusterInfo))
	if err != nil {
		t.Fatal(err)
	}
	return all[0]
}

func clusterEnv(k *fakeKube) *Env {
	env := &Env{Embedded: clusterFS, Validator: schema.New(clusterFS), Poll: 5 * time.Millisecond}
	if k != nil {
		env.Kube = k
	}
	return env
}

func TestClusterExerciseIsSkippedWithoutACluster(t *testing.T) {
	ex := clusterExercise(t)
	for name, env := range map[string]*Env{
		"no cluster support":   clusterEnv(nil),
		"cluster is not there": clusterEnv(&fakeKube{verifyErr: errors.New("no cluster")}),
	} {
		_, err := ex.RunSource(context.Background(), env, []byte(goodPod))
		if !IsSkipped(err) || !strings.Contains(err.Error(), "k8slings cluster up") {
			t.Errorf("%s: want a skip that says how to get the cluster, got %v", name, err)
		}
	}
}

func TestClusterExerciseNeedingSeveralNodes(t *testing.T) {
	ex := clusterExercise(t)
	ex.Requires = []string{"multinode"}
	_, err := ex.RunSource(context.Background(), clusterEnv(&fakeKube{nodes: 1}), []byte(goodPod))
	if !IsSkipped(err) || !strings.Contains(err.Error(), "several nodes") {
		t.Fatalf("got %v", err)
	}
	if _, err := ex.RunSource(context.Background(), clusterEnv(&fakeKube{nodes: 3}), []byte(goodPod)); err != nil {
		t.Fatalf("with 3 nodes: %v", err)
	}
}

func TestClusterExerciseWithAnUnknownAddOnIsSkipped(t *testing.T) {
	ex := clusterExercise(t)
	ex.Requires = []string{"metrics-server"}
	_, err := ex.RunSource(context.Background(), clusterEnv(&fakeKube{nodes: 3}), []byte(goodPod))
	if !IsSkipped(err) || !strings.Contains(err.Error(), "metrics-server add-on") {
		t.Fatalf("got %v", err)
	}
}

// An add-on of the registry that is not installed is skipped with the command
// that installs it, and runs once it is ready; a known add-on never says "not
// available yet".
func TestClusterExerciseNeedingAnAddOn(t *testing.T) {
	ex := clusterExercise(t)
	ex.Requires = []string{"metrics-server"}
	_, err := ex.RunSource(context.Background(), clusterEnv(&fakeKube{nodes: 3, addons: map[string]bool{"metrics-server": false}}), []byte(goodPod))
	if !IsSkipped(err) || !strings.Contains(err.Error(), "run `k8slings cluster addon install metrics-server`") {
		t.Fatalf("not installed: got %v", err)
	}
	if _, err := ex.RunSource(context.Background(), clusterEnv(&fakeKube{nodes: 3, addons: map[string]bool{"metrics-server": true}}), []byte(goodPod)); err != nil {
		t.Fatalf("installed and ready: %v", err)
	}
	// The same through the availability the list, check-all and verify use.
	av := ex.SkipReason(Availability{Cluster: true, Nodes: 3, Addons: map[string]bool{"metrics-server": false}})
	if !strings.Contains(av, "addon install metrics-server") {
		t.Errorf("SkipReason = %q", av)
	}
}

func TestStaticMistakesAreReportedWithoutTouchingTheCluster(t *testing.T) {
	k := &fakeKube{nodes: 3}
	_, err := clusterExercise(t).RunSource(context.Background(), clusterEnv(k), []byte(typoPod))
	var failed *FailedError
	if !errors.As(err, &failed) || failed.Lines[0] != "use nginx 1.27" {
		t.Fatalf("got %v", err)
	}
	if len(k.calls) != 0 {
		t.Fatalf("the cluster was touched for a static mistake: %v", k.calls)
	}
}

func TestClusterRunUsesTheExerciseNamespaceAndPollsUntilItWorks(t *testing.T) {
	k := &fakeKube{nodes: 3, readyAfter: 4}
	res, err := clusterExercise(t).RunSource(context.Background(), clusterEnv(k), []byte(goodPod))
	if err != nil || res.Exercise.Name != "web" {
		t.Fatalf("%v", err)
	}
	if k.calls[0] != "reset k8slings-web" || k.calls[1] != "apply k8slings-web" {
		t.Fatalf("the namespace is reset, then the manifest applied: %v", k.calls)
	}
	if k.reads.Load() < 4 {
		t.Fatalf("polled %d times, want at least 4", k.reads.Load())
	}
	for _, c := range k.calls[2:] {
		if !strings.HasPrefix(c, "get ") || !strings.Contains(c, "-n k8slings-web") {
			t.Fatalf("a check read outside the exercise namespace or wrote: %s", c)
		}
	}
}

func TestClusterRunReportsTheProblemAndDiagnosisOnTimeout(t *testing.T) {
	k := &fakeKube{nodes: 3, readyAfter: 1 << 30}
	_, err := clusterExercise(t).RunSource(context.Background(), clusterEnv(k), []byte(goodPod))
	var failed *FailedError
	if !errors.As(err, &failed) {
		t.Fatalf("got %v", err)
	}
	if len(failed.Lines) != 1 || failed.Lines[0] != "the Pod is not Running yet" {
		t.Fatalf("the problem: %v", failed.Lines)
	}
	if got := strings.Join(failed.Details, "\n"); !strings.Contains(got, "web  Pending  ready 0/1") {
		t.Fatalf("the cluster state should be in the details, not the problems:\n%s", got)
	}
}

func TestEnvTimeoutOverridesTheExercise(t *testing.T) {
	k := &fakeKube{nodes: 3, readyAfter: 1 << 30}
	env := clusterEnv(k)
	env.Timeout = 30 * time.Millisecond
	start := time.Now()
	_, err := clusterExercise(t).RunSource(context.Background(), env, []byte(goodPod))
	if err == nil || time.Since(start) > 300*time.Millisecond {
		t.Fatalf("a short override should end the wait early (took %v): %v", time.Since(start), err)
	}
}

func TestApplyRejectionIsReportedAsTheMistake(t *testing.T) {
	k := &fakeKube{nodes: 3, applyErr: true, applyOut: "Error from server (Invalid): Deployment \"web\" is invalid: selector does not match template labels\nsomething else"}
	_, err := clusterExercise(t).RunSource(context.Background(), clusterEnv(k), []byte(goodPod))
	var failed *FailedError
	if !errors.As(err, &failed) || len(failed.Lines) != 1 || !strings.Contains(failed.Lines[0], "selector does not match template labels") {
		t.Fatalf("got %v", err)
	}
	if k.reads.Load() != 0 {
		t.Fatal("nothing should be polled after a rejected apply")
	}
}

func TestCancelledRunStopsWaiting(t *testing.T) {
	k := &fakeKube{nodes: 3, readyAfter: 1 << 30}
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(30 * time.Millisecond); cancel() }()
	env := clusterEnv(k)
	env.Timeout = time.Minute
	start := time.Now()
	_, err := clusterExercise(t).RunSource(ctx, env, []byte(goodPod))
	if err == nil || time.Since(start) > 2*time.Second {
		t.Fatalf("cancel should stop the wait (took %v): %v", time.Since(start), err)
	}
}

func TestNextPendingSkipsClusterExercisesWithoutACluster(t *testing.T) {
	t.Chdir(t.TempDir())
	info := clusterInfo + "[[exercises]]\nname='plain'\npath='exercises/t/plain'\ntopic='t'\nmode='manifest'\nhint='h'\n"
	if err := writeFile("info.toml", info); err != nil {
		t.Fatal(err)
	}
	ex, err := NextPending("info.toml", Availability{})
	if err != nil || ex.Name != "plain" {
		t.Fatalf("without a cluster the next one is the manifest exercise, got %v %v", ex.Name, err)
	}
	ex, err = NextPending("info.toml", Availability{Cluster: true, Nodes: 3})
	if err != nil || ex.Name != "web" {
		t.Fatalf("with a cluster the cluster exercise comes first, got %v %v", ex.Name, err)
	}
}

func TestFinalProblemsDoNotWaitForTheTimeout(t *testing.T) {
	fs := fstest.MapFS{"exercises/t/web/check.yaml": {Data: []byte(`
schema: off
steps:
  - assert: {resource: pod/web, expr: "object.status.phase == 'Nope'", message: "wrong on purpose", now: true}
`)}}
	env := clusterEnv(&fakeKube{nodes: 3, readyAfter: 1})
	env.Embedded = fs
	env.Timeout = time.Minute
	start := time.Now()
	_, err := clusterExercise(t).RunSource(context.Background(), env, []byte(goodPod))
	var failed *FailedError
	if !errors.As(err, &failed) || failed.Lines[0] != "wrong on purpose" || time.Since(start) > time.Second {
		t.Fatalf("took %v: %v", time.Since(start), err)
	}
}

// MissingAddons names what to install, SkipReason says it, and SkippedForAddons
// counts the exercises that only wait for an add-on.
func TestMissingAddonsAreNamedAllTogether(t *testing.T) {
	av := Availability{Cluster: true, Nodes: 3, Addons: map[string]bool{"gateway": false, "metrics-server": false, "other": true}}
	both := Exercise{Name: "both", Mode: ModeCluster, Requires: []string{"metrics-server", "gateway", "other"}}
	if got := both.MissingAddons(av); len(got) != 2 || got[0] != "metrics-server" || got[1] != "gateway" {
		t.Errorf("MissingAddons = %v", got)
	}
	if r := both.SkipReason(av); !strings.Contains(r, "needs the metrics-server add-on") || !strings.HasSuffix(r, "`k8slings cluster addon install metrics-server gateway`") {
		t.Errorf("SkipReason = %q", r)
	}

	// Something else comes first: the cluster, then the nodes. Then the add-on is
	// not what to install.
	if got := both.MissingAddons(Availability{Addons: av.Addons}); got != nil {
		t.Errorf("with no cluster the cluster is the cause, got %v", got)
	}
	multi := Exercise{Name: "m", Mode: ModeCluster, Requires: []string{"multinode", "gateway"}}
	if got := multi.MissingAddons(Availability{Cluster: true, Nodes: 1, Addons: av.Addons}); got != nil {
		t.Errorf("several nodes are needed first, got %v", got)
	}
	if got := (Exercise{Mode: ModeCluster, Requires: []string{"unheard-of"}}).MissingAddons(av); got != nil {
		t.Errorf("an add-on this build does not know cannot be installed, got %v", got)
	}

	skipped := []Exercise{both, {Name: "g", Mode: ModeCluster, Requires: []string{"gateway"}}, {Name: "x", Mode: ModeCluster, Requires: []string{"unheard-of"}}}
	n, names := SkippedForAddons(skipped, av)
	if n != 2 || len(names) != 2 || names[0] != "gateway" || names[1] != "metrics-server" {
		t.Errorf("SkippedForAddons = %d %v (sorted, once each, only the exercises that wait for an add-on)", n, names)
	}
}
