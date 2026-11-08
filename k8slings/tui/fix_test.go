package tui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/cleonte/k8slings/k8slings/cluster"
	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/cleonte/k8slings/k8slings/schema"
)

func init() { fixRecheckEvery = time.Millisecond }

// fakeCluster is a cluster that is always there and healthy, and counts seeds.
type fakeCluster struct{ seeds, seededCalls atomic.Int32 }

func (f *fakeCluster) Kubectl(context.Context, ...string) (string, error) { return "", nil }
func (f *fakeCluster) Verify(context.Context) error                       { return nil }
func (f *fakeCluster) ResetNamespace(context.Context, string) error       { return nil }
func (f *fakeCluster) Apply(context.Context, string, []byte) (string, error) {
	return "", nil
}
func (f *fakeCluster) Delete(context.Context, string, []byte) error { return nil }
func (f *fakeCluster) Query(context.Context, []string, bool) (cluster.QueryResult, error) {
	return cluster.QueryResult{}, nil
}
func (f *fakeCluster) Nodes(context.Context) (int, error)        { return 3, nil }
func (f *fakeCluster) Diagnose(context.Context, string) []string { return nil }
func (f *fakeCluster) Seeded(context.Context, string, string) bool {
	f.seededCalls.Add(1)
	return false
}
func (f *fakeCluster) UID(context.Context, string, string) (string, error) {
	return "uid", nil
}
func (f *fakeCluster) Seed(context.Context, string, string, [][]byte) error {
	f.seeds.Add(1)
	return nil
}

const fixInfo = "[[exercises]]\nname='f1'\npath='exercises/t/f1'\ntopic='t'\nmode='fix'\nhint='Look at the logs.'\ndocs='https://kubernetes.io'\n"

// fixModel is a model over one fix exercise, whose checks are answered by run.
func fixModel(t *testing.T, run RunFunc) (*Model, *fakeCluster) {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.WriteFile("info.toml", []byte(fixInfo), 0o644); err != nil {
		t.Fatal(err)
	}
	embedded := fstest.MapFS{
		"exercises/t/f1/setup.yaml": {Data: []byte("kind: Pod\n")},
		"exercises/t/f1/check.yaml": {Data: []byte("shipped: \"x\"\nsteps:\n  - logs: {pod: a, contains: b}\n")},
	}
	kube := &fakeCluster{}
	m := New(Config{
		InfoFile: "info.toml",
		Env:      &exercises.Env{Embedded: embedded, Validator: schema.New(embedded), Kube: kube},
		Run:      run,
	})
	m.width, m.height = 100, 30
	settle(t, m, m.startRun())
	return m, kube
}

func seededRun(context.Context, exercises.Exercise) (exercises.Result, error) {
	return exercises.Result{}, &exercises.SeededError{Exercise: exercises.Exercise{Name: "f1"}, Namespace: "k8slings-f1"}
}

func failingRun(context.Context, exercises.Exercise) (exercises.Result, error) {
	return exercises.Result{}, &exercises.FailedError{Lines: []string{"Deployment web: 0 of 2 replicas are available."}}
}

func TestSeededScreenHandsOverToTheLearner(t *testing.T) {
	m, _ := fixModel(t, seededRun)
	// The path is shown the way the platform writes it: backslashes on Windows.
	want(t, m, "the cluster is now broken on purpose", filepath.FromSlash("exercises/t/f1/README.md"),
		"k8slings kubectl -n k8slings-f1 get pods", "every couple of seconds")
	if strings.Contains(screen(m), "not solved yet") {
		t.Fatal("being seeded is not a failure")
	}
}

func TestFailureOfAFixExerciseSaysHowToRepairIt(t *testing.T) {
	m, _ := fixModel(t, failingRun)
	want(t, m, "0 of 2 replicas are available", "Repair it with k8slings kubectl -n k8slings-f1")
	if strings.Contains(screen(m), "Edit exercises") {
		t.Fatal("there is no file to edit")
	}
}

func TestAFixExerciseIsCheckedAgainEveryFewSeconds(t *testing.T) {
	var runs atomic.Int32
	m, _ := fixModel(t, func(ctx context.Context, ex exercises.Exercise) (exercises.Result, error) {
		runs.Add(1)
		return failingRun(ctx, ex)
	})
	first := runs.Load()
	if m.scheduleFixRecheck() == nil {
		t.Fatal("a fix exercise that is not repaired must be looked at again")
	}

	// The tick comes: another look.
	_, cmd := m.Update(fixTickMsg{id: m.runID})
	settle(t, m, cmd)
	if runs.Load() != first+1 {
		t.Fatalf("runs = %d, want %d", runs.Load(), first+1)
	}

	// An old tick (the run it was for is over) does nothing.
	old := m.runID - 1
	_, cmd = m.Update(fixTickMsg{id: old})
	if cmd != nil || runs.Load() != first+1 {
		t.Fatal("a stale tick must be ignored")
	}
}

func TestTheRecheckWaitsWhileTheLearnerReadsTheHint(t *testing.T) {
	var runs atomic.Int32
	m, _ := fixModel(t, func(ctx context.Context, ex exercises.Exercise) (exercises.Result, error) {
		runs.Add(1)
		return failingRun(ctx, ex)
	})
	press(t, m, "h")
	want(t, m, "Hint for f1", "Look at the logs.")
	before := runs.Load()
	_, cmd := m.Update(fixTickMsg{id: m.runID})
	if cmd == nil {
		t.Fatal("the recheck must keep going, or it stops for good after the hint")
	}
	if runs.Load() != before || m.mode != modeHint {
		t.Fatal("reading the hint must not be interrupted by a new run")
	}
}

func TestARepairedFixExerciseStopsBeingChecked(t *testing.T) {
	m, _ := fixModel(t, func(context.Context, exercises.Exercise) (exercises.Result, error) {
		return exercises.Result{}, nil
	})
	want(t, m, "f1 works!", "The cluster is repaired")
	if m.scheduleFixRecheck() != nil {
		t.Fatal("nothing left to look at")
	}
}

func TestASkippedFixExerciseIsNotRechecked(t *testing.T) {
	m, _ := fixModel(t, func(context.Context, exercises.Exercise) (exercises.Result, error) {
		return exercises.Result{}, &exercises.SkippedError{Reason: "f1 needs the cluster"}
	})
	if m.scheduleFixRecheck() != nil {
		t.Fatal("a skipped exercise has nothing to wait for")
	}
}

func TestResetBreaksTheClusterAgainAfterAskingFirst(t *testing.T) {
	m, kube := fixModel(t, failingRun)
	press(t, m, "r")
	want(t, m, "Reset f1 to its original state?", "breaks the cluster again")
	seedsBefore := kube.seeds.Load()
	press(t, m, "y")
	if kube.seeds.Load() != seedsBefore+1 {
		t.Fatalf("seeds = %d, want %d", kube.seeds.Load(), seedsBefore+1)
	}
	want(t, m, "Reset f1: the cluster is broken again.")
}

func TestResetFailureIsReported(t *testing.T) {
	m, _ := fixModel(t, failingRun)
	_, cmd := m.Update(reseedDoneMsg{err: errors.New("the cluster went away")})
	if cmd != nil {
		t.Fatal("nothing more to do after a failed reset")
	}
	want(t, m, "Reset failed: the cluster went away")
}

// Checking everything must never break the cluster for an exercise the learner
// has not started: that would wipe nothing yet, but it would start the exercise
// behind their back.
func TestCheckAllNeverSeedsAFixExercise(t *testing.T) {
	m, kube := fixModel(t, nil) // the real Run, so that the real guard is what is tested
	m.cfg.Run = func(ctx context.Context, ex exercises.Exercise) (exercises.Result, error) {
		return ex.Run(ctx, m.cfg.Env)
	}
	seeds := kube.seeds.Load() // the first view seeded it, as it should
	press(t, m, "c")
	if kube.seeds.Load() != seeds {
		t.Fatalf("check-all seeded: %d -> %d", seeds, kube.seeds.Load())
	}
	want(t, m, "• f1", "has not been started")
}

func TestKeysForExercisesWithoutAFileDoNotCrash(t *testing.T) {
	m, _ := fixModel(t, failingRun)
	for _, k := range []string{"h", "esc", "f", "n", "l", "esc", "c", "esc"} {
		press(t, m, k)
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if !isQuit(cmd) {
		t.Fatal("q should still quit")
	}
}

// Setting up a fix exercise can take a minute: the screen says which step it is
// on, and quitting cancels the setup, so that nothing is kept.
func TestSetupShowsProgressAndQuittingCancelsIt(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("info.toml", []byte(fixInfo), 0o644); err != nil {
		t.Fatal(err)
	}
	embedded := fstest.MapFS{"exercises/t/f1/setup.yaml": {Data: []byte("kind: Pod\n")}}
	started, cancelled := make(chan struct{}), make(chan error, 1)
	m := New(Config{
		InfoFile: "info.toml",
		Env:      &exercises.Env{Embedded: embedded, Validator: schema.New(embedded), Kube: &fakeCluster{}},
		Run: func(ctx context.Context, ex exercises.Exercise) (exercises.Result, error) {
			cluster.Progress(ctx, "setting up f1: step 1 of 2")
			close(started)
			<-ctx.Done()
			cancelled <- ctx.Err()
			return exercises.Result{}, ctx.Err()
		},
	})
	m.width, m.height = 100, 30
	cmd := m.startRun()
	go cmd()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("the run never started")
	}
	want(t, m, "setting up f1: step 1 of 2", "nothing is kept")

	press(t, m, "q")
	select {
	case err := <-cancelled:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("the setup was stopped with %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("quitting did not cancel the setup")
	}
}
