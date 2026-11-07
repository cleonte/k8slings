package exercises

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/cleonte/k8slings/k8slings/schema"
)

const fixInfo = `[[exercises]]
name='web'
path='exercises/t/web'
topic='t'
mode='fix'
hint='h'
`

var fixFS = fstest.MapFS{
	"exercises/t/web/setup.yaml": {Data: []byte("kind: Pod\nmetadata: {name: web}\n")},
	"exercises/t/web/check.yaml": {Data: []byte(`
shipped: "not Running"
steps:
  - unchanged: pod/web
  - assert: {resource: pod/web, expr: "object.status.phase == 'Running'", message: "the Pod is not Running yet"}
`)},
}

func fixExercise(t *testing.T) Exercise {
	t.Helper()
	t.Chdir(t.TempDir()) // the progress file lives in the working directory
	all, err := ParseInfo([]byte(fixInfo))
	if err != nil {
		t.Fatal(err)
	}
	return all[0]
}

func fixEnv(k *fakeKube) *Env {
	env := &Env{Embedded: fixFS, Validator: schema.New(fixFS), Settle: time.Millisecond}
	if k != nil { // a nil *fakeKube in the interface would not be a nil Kube
		env.Kube = k
	}
	return env
}

func TestFixSeedsOnTheFirstRunAndSaysSoInsteadOfFailing(t *testing.T) {
	ex := fixExercise(t)
	k := &fakeKube{nodes: 3}
	_, err := ex.Run(context.Background(), fixEnv(k))
	if !IsSeeded(err) {
		t.Fatalf("the first run should seed, got %v", err)
	}
	var failed *FailedError
	if errors.As(err, &failed) {
		t.Fatal("being seeded is not a failure")
	}
	if k.seedCalls != 1 || !strings.Contains(k.seedSetup, "name: web") {
		t.Fatalf("seed calls = %d, setup = %q", k.seedCalls, k.seedSetup)
	}
	if ex.State() != Pending {
		t.Fatal("a seeded exercise is not done")
	}
	var seeded *SeededError
	if !errors.As(err, &seeded) || seeded.Namespace != "k8slings-web" {
		t.Fatalf("the notice should name the namespace: %v", err)
	}
}

func TestFixNeverSeedsAgainOnItsOwn(t *testing.T) {
	ex := fixExercise(t)
	k := &fakeKube{nodes: 3, seeded: true, readyAfter: 1 << 30}
	if err := recordSeed("web", map[string]string{"pod/web": "uid-pod/web"}); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		_, err := ex.Run(context.Background(), fixEnv(k))
		var failed *FailedError
		if !errors.As(err, &failed) || failed.Lines[0] != "the Pod is not Running yet" {
			t.Fatalf("got %v", err)
		}
	}
	if k.seedCalls != 0 {
		t.Fatalf("seeded %d times: that would wipe the learner's repair", k.seedCalls)
	}
}

func TestFixIsRememberedAsDoneOnceItPasses(t *testing.T) {
	ex := fixExercise(t)
	k := &fakeKube{nodes: 3, seeded: true, readyAfter: 1}
	_ = recordSeed("web", map[string]string{"pod/web": "uid-pod/web"})
	if _, err := ex.Run(context.Background(), fixEnv(k)); err != nil {
		t.Fatal(err)
	}
	if ex.State() != Done {
		t.Fatal("a repaired exercise is done")
	}
	data, err := os.ReadFile(ProgressFile)
	if err != nil || !strings.Contains(string(data), `"web"`) {
		t.Fatalf("the progress file should name it: %s %v", data, err)
	}

	// Done is sticky: breaking it again later does not undo it.
	k.readyAfter = 1 << 30
	k.reads.Store(0)
	if _, err := ex.Run(context.Background(), fixEnv(k)); err == nil {
		t.Fatal("it is broken again, so the check fails")
	}
	if ex.State() != Done {
		t.Fatal("once done, it stays done")
	}
	if _, err := NextPending("info.toml", Availability{Cluster: true, Nodes: 3}); err == nil {
		// info.toml is not written in this test; the point is State above
		t.Log("(no info.toml here)")
	}
}

// A crashing container is Running and Ready for a moment before it exits. One
// look that lands on that moment must not call the exercise fixed.
func TestFixMustHoldAfterAMoment(t *testing.T) {
	ex := fixExercise(t)
	k := &fakeKube{nodes: 3, seeded: true, flap: true}
	_ = recordSeed("web", map[string]string{"pod/web": "uid-pod/web"})
	_, err := ex.Run(context.Background(), fixEnv(k))
	var failed *FailedError
	if !errors.As(err, &failed) || !strings.Contains(failed.Lines[0], "did not hold") {
		t.Fatalf("a repair that does not hold must fail, got %v", err)
	}
	if ex.State() != Pending {
		t.Fatal("a flapping cluster must not be recorded as fixed")
	}
}

func TestFixDeletingAndRecreatingIsNotARepair(t *testing.T) {
	ex := fixExercise(t)
	k := &fakeKube{nodes: 3, seeded: true, readyAfter: 1, liveUID: "a-brand-new-uid"}
	_ = recordSeed("web", map[string]string{"pod/web": "uid-pod/web"})
	_, err := ex.Run(context.Background(), fixEnv(k))
	var failed *FailedError
	if !errors.As(err, &failed) || !strings.Contains(failed.Lines[0], "deleted and created again") {
		t.Fatalf("got %v", err)
	}
	if ex.State() != Pending {
		t.Fatal("a shortcut must not count")
	}
}

// A namespace left over from another folder, or from before the progress file
// was lost, cannot be judged: nobody knows which objects were the original ones.
// Starting the exercise fresh beats leaving the learner stuck.
func TestFixStartsFreshWhenThisWorkspaceDidNotSeedIt(t *testing.T) {
	ex := fixExercise(t)
	k := &fakeKube{nodes: 3, seeded: true, readyAfter: 1} // seeded by someone else
	_, err := ex.Run(context.Background(), fixEnv(k))
	if !IsSeeded(err) || k.seedCalls != 1 {
		t.Fatalf("it should be seeded again, got %v (seeds: %d)", err, k.seedCalls)
	}

	// ...but checking everything still must not touch it. (A new folder: the
	// first half recorded the seed in this one.)
	t.Chdir(t.TempDir())
	k2 := &fakeKube{nodes: 3, seeded: true}
	_, err = ex.Run(NoSeed(context.Background()), fixEnv(k2))
	if !IsSkipped(err) || k2.seedCalls != 0 {
		t.Fatalf("got %v (seeds: %d)", err, k2.seedCalls)
	}
}

func TestNoSeedContextNeverBreaksTheCluster(t *testing.T) {
	ex := fixExercise(t)
	k := &fakeKube{nodes: 3}
	_, err := ex.Run(NoSeed(context.Background()), fixEnv(k))
	if !IsSkipped(err) || !strings.Contains(err.Error(), "has not been started") {
		t.Fatalf("checking everything must skip a fix exercise that was not started, got %v", err)
	}
	if k.seedCalls != 0 {
		t.Fatal("verify and check-all must not seed")
	}
}

func TestReseedBreaksTheClusterAgainAndForgetsDone(t *testing.T) {
	ex := fixExercise(t)
	_ = markPassed("web")
	if ex.State() != Done {
		t.Fatal("setup")
	}
	k := &fakeKube{nodes: 3, seeded: true}
	if err := ex.Reseed(context.Background(), fixEnv(k)); err != nil {
		t.Fatal(err)
	}
	if k.seedCalls != 1 {
		t.Fatalf("seed calls = %d", k.seedCalls)
	}
	if ex.State() != Pending {
		t.Fatal("a reset starts the exercise over")
	}
	if uid, ok := seedUID("web", "pod/web"); !ok || uid != "uid-pod/web" {
		t.Fatalf("the seeded UID should be recorded again: %q %v", uid, ok)
	}
}

func TestFixNeedsTheCluster(t *testing.T) {
	ex := fixExercise(t)
	_, err := ex.Run(context.Background(), fixEnv(nil))
	if !IsSkipped(err) || !strings.Contains(err.Error(), "k8slings cluster up") {
		t.Fatalf("got %v", err)
	}
	if reason := ex.SkipReason(Availability{}); !strings.Contains(reason, "cluster up") {
		t.Fatalf("reason = %q", reason)
	}
	if reason := ex.SkipReason(Availability{Cluster: true, Nodes: 3}); reason != "" {
		t.Fatalf("with a cluster it can run, got %q", reason)
	}
}

func TestFixExerciseHasNoFileToEditAndItsTicketIsTheReadme(t *testing.T) {
	ex := fixExercise(t)
	if !strings.HasSuffix(ex.File(), "README.md") {
		t.Fatalf("File() = %s", ex.File())
	}
	if !strings.HasSuffix(ex.SolutionPath(), "solution.sh") {
		t.Fatalf("SolutionPath() = %s", ex.SolutionPath())
	}
	if err := ex.MarkDone(); err != nil {
		t.Fatalf("MarkDone is a no-op for a fix exercise: %v", err)
	}
}

func TestADamagedProgressFileMeansNoProgressNotACrash(t *testing.T) {
	ex := fixExercise(t)
	if err := os.WriteFile(ProgressFile, []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ex.State() != Pending {
		t.Fatal("a damaged file means nothing is done")
	}
	if err := markPassed("web"); err != nil {
		t.Fatal(err)
	}
	if ex.State() != Done {
		t.Fatal("and it is written again")
	}
}

func TestProgressWritesAreAtomicAndSafeToRaceOn(t *testing.T) {
	ex := fixExercise(t)
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if i%2 == 0 {
				_ = markPassed("web")
			} else {
				_ = recordSeed("other", map[string]string{"x/y": "u"})
			}
			_ = ex.State()
		}()
	}
	wg.Wait()
	data, err := os.ReadFile(ProgressFile)
	if err != nil {
		t.Fatal(err)
	}
	var p progressData
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("the file must always be valid JSON: %v\n%s", err, data)
	}
	if left, _ := os.ReadDir("."); len(left) != 1 {
		t.Fatalf("temporary files were left behind: %v", left)
	}
	_ = time.Now
}

func TestFixSeedsEveryStepInOrder(t *testing.T) {
	ex := fixExercise(t)
	fsys := fstest.MapFS{
		"exercises/t/web/setup.yaml":   {Data: []byte("one")},
		"exercises/t/web/setup-2.yaml": {Data: []byte("two")},
		"exercises/t/web/check.yaml":   fixFS["exercises/t/web/check.yaml"],
	}
	k := &fakeKube{nodes: 3}
	env := &Env{Embedded: fsys, Validator: schema.New(fsys), Settle: time.Millisecond, Kube: k}
	if _, err := ex.Run(context.Background(), env); !IsSeeded(err) {
		t.Fatalf("the first run should seed, got %v", err)
	}
	if k.seedSetup != "one|two" {
		t.Fatalf("the steps must reach Seed in order, got %q", k.seedSetup)
	}
}

// A seeding that fails leaves the namespace labelled but records no UIDs, so the
// next run seeds again from scratch instead of judging a half-built cluster.
func TestFixFailedSeedingIsStartedOverByTheNextRun(t *testing.T) {
	ex := fixExercise(t)
	k := &fakeKube{nodes: 3, seedErr: errors.New("cannot set up web: step 1 of 2 did not finish rolling out")}
	_, err := ex.Run(context.Background(), fixEnv(k))
	if err == nil || IsSeeded(err) || !strings.Contains(err.Error(), "step 1 of 2") {
		t.Fatalf("a failed seeding is an error, not a hand-over: %v", err)
	}
	if !k.seeded {
		t.Fatal("the fake should have kept the label, as the real seeding does")
	}
	if hasSeed("web") {
		t.Fatal("nothing may be recorded for a seeding that failed")
	}

	k.seedErr = nil
	if _, err := ex.Run(context.Background(), fixEnv(k)); !IsSeeded(err) {
		t.Fatalf("the next run should seed again, got %v", err)
	}
	if k.seedCalls != 2 {
		t.Fatalf("seeded %d times, want 2", k.seedCalls)
	}
	if !hasSeed("web") {
		t.Fatal("the second seeding finished and must be recorded")
	}
}

// Seeding again over an earlier seeding must not leave the old UIDs behind when
// it fails: they would make the half-built cluster look seeded.
func TestFixFailedReseedForgetsTheEarlierSeeding(t *testing.T) {
	ex := fixExercise(t)
	k := &fakeKube{nodes: 3}
	if _, err := ex.Run(context.Background(), fixEnv(k)); !IsSeeded(err) {
		t.Fatalf("setup: %v", err)
	}
	k.seedErr = errors.New("boom")
	if err := ex.Reseed(context.Background(), fixEnv(k)); err == nil {
		t.Fatal("the reseed should fail")
	}
	if hasSeed("web") {
		t.Fatal("a failed reseed must not keep the UIDs of the earlier seeding")
	}
}

func TestFixCancelledSeedingRecordsNothingAndTheNextRunStartsOver(t *testing.T) {
	ex := fixExercise(t)
	k := &fakeKube{nodes: 3, seedBlock: true}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	_, err := ex.Run(ctx, fixEnv(k))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelling mid-seed returns the context error, got %v", err)
	}
	if IsSeeded(err) {
		t.Fatal("a cancelled seeding is not a hand-over")
	}
	if hasSeed("web") || ex.State() != Pending {
		t.Fatal("a cancelled seeding must record nothing")
	}

	k.seedBlock = false
	if _, err := ex.Run(context.Background(), fixEnv(k)); !IsSeeded(err) {
		t.Fatalf("the next run should start over, got %v", err)
	}
	if k.seedCalls != 2 {
		t.Fatalf("seeded %d times, want 2", k.seedCalls)
	}
}

func TestEmbeddedOnlyFiles(t *testing.T) {
	for name, want := range map[string]bool{
		"check.yaml": true, "setup.yaml": true, "setup-2.yaml": true, "setup-10.yaml": true,
		"exercises/x/y/setup-3.yaml": true, "README.md": false, "exercise.yaml": false,
		"setup-x.yaml": false, "mysetup-2.yaml": false, "setup-2.yaml.bak": false,
	} {
		if got := EmbeddedOnly(name); got != want {
			t.Errorf("EmbeddedOnly(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestSetupFilesAreConsecutiveAndCapped(t *testing.T) {
	mk := func(names ...string) fstest.MapFS {
		m := fstest.MapFS{}
		for _, n := range names {
			m["d/"+n] = &fstest.MapFile{Data: []byte("x")}
		}
		return m
	}
	if got, err := SetupFiles(mk("setup.yaml", "setup-2.yaml", "setup-3.yaml"), "d"); err != nil || len(got) != 3 || got[2] != "d/setup-3.yaml" {
		t.Fatalf("got %v, %v", got, err)
	}
	if got, err := SetupFiles(mk("setup.yaml"), "d"); err != nil || len(got) != 1 {
		t.Fatalf("one step: %v, %v", got, err)
	}
	for name, fsys := range map[string]fstest.MapFS{
		"no first step": mk("setup-2.yaml"),
		"a gap":         mk("setup.yaml", "setup-3.yaml"),
		"too many":      mk("setup.yaml", "setup-2.yaml", "setup-3.yaml", "setup-4.yaml", "setup-5.yaml", "setup-6.yaml"),
	} {
		if _, err := SetupFiles(fsys, "d"); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
