package tui

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/cleonte/k8slings/k8slings/exercises"
)

// runningLine matches the line of an exercise that is being checked (not the
// title, which also contains an ellipsis).
var runningLine = regexp.MustCompile(`(?m)^… [a-z]`)

// gatedRun is a Run that blocks until the test lets an exercise finish, and
// records how many were running at the same time.
type gatedRun struct {
	mu        sync.Mutex
	gates     map[string]chan error
	started   atomic.Int32
	running   atomic.Int32
	maxActive atomic.Int32
	cancelled atomic.Int32
}

func newGatedRun() *gatedRun { return &gatedRun{gates: map[string]chan error{}} }

func (g *gatedRun) gate(name string) chan error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.gates[name] == nil {
		g.gates[name] = make(chan error, 1)
	}
	return g.gates[name]
}

func (g *gatedRun) run(ctx context.Context, ex exercises.Exercise) (exercises.Result, error) {
	g.started.Add(1)
	n := g.running.Add(1)
	for {
		old := g.maxActive.Load()
		if n <= old || g.maxActive.CompareAndSwap(old, n) {
			break
		}
	}
	defer g.running.Add(-1)
	select {
	case err := <-g.gate(ex.Name):
		return exercises.Result{}, err
	case <-ctx.Done():
		g.cancelled.Add(1)
		return exercises.Result{}, ctx.Err()
	}
}

// pumpBatch starts the commands of a (possibly batched) command in goroutines,
// like Bubble Tea does, and returns the channel their messages arrive on.
func pumpBatch(cmd tea.Cmd) <-chan tea.Msg {
	out := make(chan tea.Msg, 64)
	var start func(c tea.Cmd)
	start = func(c tea.Cmd) {
		if c == nil {
			return
		}
		go func() {
			msg := c()
			if batch, ok := msg.(tea.BatchMsg); ok {
				for _, b := range batch {
					start(b)
				}
				return
			}
			out <- msg
		}()
	}
	start(cmd)
	return out
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func gatedModel(t *testing.T) (*Model, *gatedRun) {
	t.Helper()
	m := newModel(t)
	g := newGatedRun()
	m.cfg.Run = g.run
	return m, g
}

func TestCheckAllRunsAFewAtATimeAndShowsProgress(t *testing.T) {
	m, g := gatedModel(t)
	cmd := m.startCheck()
	out := pumpBatch(cmd)

	waitFor(t, "the first workers to start", func() bool { return g.running.Load() == checkWorkers })
	want(t, m, "Checking the exercises… 0/")
	if got := len(runningLine.FindAllString(screen(m), -1)); got != checkWorkers {
		t.Fatalf("%d exercises show as running, want %d:\n%s", got, checkWorkers, screen(m))
	}
	if !strings.Contains(screen(m), "· ") {
		t.Fatalf("the rest should be waiting:\n%s", screen(m))
	}

	// One finishes: the next starts, and the counter moves.
	first := m.checkItems[0].exercise.Name
	g.gate(first) <- nil
	_, next := m.Update(<-out)
	_ = pumpBatch(next)
	waitFor(t, "the next exercise to start", func() bool { return g.started.Load() > checkWorkers })
	want(t, m, "Checking the exercises… 1/")
	if max := g.maxActive.Load(); max > checkWorkers {
		t.Fatalf("%d exercises ran at once, the limit is %d", max, checkWorkers)
	}
	m.stopCheck() // release the goroutines
}

func TestAnotherCWhileCheckingDoesNotStartASecondCheck(t *testing.T) {
	m, g := gatedModel(t)
	_ = pumpBatch(m.startCheck())
	waitFor(t, "workers", func() bool { return g.running.Load() == checkWorkers })
	started := g.started.Load()

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	if cmd != nil {
		t.Fatal("a second c must not start anything")
	}
	want(t, m, "Already checking the exercises.")
	time.Sleep(50 * time.Millisecond)
	if g.started.Load() != started {
		t.Fatalf("a second check started exercises: %d -> %d", started, g.started.Load())
	}
	m.stopCheck()
}

func TestEscapeCancelsTheCheckAndIgnoresLateResults(t *testing.T) {
	m, g := gatedModel(t)
	out := pumpBatch(m.startCheck())
	waitFor(t, "workers", func() bool { return g.running.Load() == checkWorkers })
	oldID := m.checkID

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.checking() {
		t.Fatal("esc should cancel the check")
	}
	waitFor(t, "the running exercises to be cancelled", func() bool { return g.cancelled.Load() == checkWorkers })
	_ = cmd // the exercise is run again; not needed here

	// What the cancelled workers report arrives late, and must change nothing.
	before := screen(m)
	for range checkWorkers {
		select {
		case msg := <-out:
			if done, ok := msg.(checkDoneMsg); ok && done.id != oldID {
				t.Fatalf("a result carries id %d, want the old id %d", done.id, oldID)
			}
			m.Update(msg)
		case <-time.After(2 * time.Second):
			t.Fatal("the cancelled workers did not report")
		}
	}
	if m.checking() {
		t.Fatal("late results must not revive a cancelled check")
	}
	if after := screen(m); after != before {
		t.Fatalf("late results changed the screen:\n%s\n---\n%s", before, after)
	}
}

func TestASaveDoesNotThrowAwayACheckInProgress(t *testing.T) {
	m, g := gatedModel(t)
	_ = pumpBatch(m.startCheck())
	waitFor(t, "workers", func() bool { return g.running.Load() == checkWorkers })

	_, cmd := m.Update(changesMsg{files: []string{"exercises/pods/pods1/exercise.yaml"}})
	if !m.checking() || m.mode != modeCheck {
		t.Fatal("a save must leave the check alone")
	}
	_ = cmd
	m.stopCheck()
}

func TestSkippedExercisesShowWhy(t *testing.T) {
	m, g := gatedModel(t)
	out := pumpBatch(m.startCheck())
	waitFor(t, "workers", func() bool { return g.running.Load() == checkWorkers })
	g.gate(m.checkItems[0].exercise.Name) <- &exercises.SkippedError{Reason: "needs the cluster: run `k8slings cluster up`"}
	m.Update(<-out)
	want(t, m, "• "+m.checkItems[0].exercise.Name, "needs the cluster")
	if m.checkItems[0].status != checkSkipped {
		t.Fatalf("status = %d", m.checkItems[0].status)
	}
	m.stopCheck()
}

func TestFailureShowsItsFirstProblemAndAFinishedOneIsFlagged(t *testing.T) {
	m := newModel(t)
	solve(t, "pods1", false) // done, and working
	// Break it again, keeping the marker out: finished, but now failing.
	ex, _ := exercises.Find("pods1", "info.toml")
	if err := writeFile(ex.File(), "apiVersion: v1\nkind: Pod\nmetadata: {name: hello}\nspec: {contianers: []}\n"); err != nil {
		t.Fatal(err)
	}
	press(t, m, "c")
	want(t, m, "✗ pods1", "was finished, but now fails")
}

func TestAPassingExerciseWithItsMarkerSaysSo(t *testing.T) {
	m := newModel(t)
	solve(t, "pods1", true)
	press(t, m, "c")
	want(t, m, "✔ pods1", "the I AM NOT DONE marker is still there")
}

func TestErrorFromRunIsAFailureNotAPanic(t *testing.T) {
	m, g := gatedModel(t)
	out := pumpBatch(m.startCheck())
	waitFor(t, "workers", func() bool { return g.running.Load() == checkWorkers })
	g.gate(m.checkItems[0].exercise.Name) <- errors.New("cannot read the file")
	m.Update(<-out)
	want(t, m, "✗ "+m.checkItems[0].exercise.Name, "cannot read the file")
	m.stopCheck()
}

func TestQuitStopsACheckInProgress(t *testing.T) {
	m, g := gatedModel(t)
	_ = pumpBatch(m.startCheck())
	waitFor(t, "workers", func() bool { return g.running.Load() == checkWorkers })
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Fatal("q should quit")
	}
	waitFor(t, "the workers to be cancelled", func() bool { return g.cancelled.Load() == checkWorkers })
}
