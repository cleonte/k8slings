package tui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/cleonte/k8slings"
	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/cleonte/k8slings/k8slings/schema"
	"github.com/cleonte/k8slings/k8slings/workspace"
)

// newModel starts a model in a fresh workspace made from the embedded exercises.
func newModel(t *testing.T) *Model {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "ws")
	if err := workspace.Init(k8slings.FS, dir); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	m := New(Config{
		InfoFile:  "info.toml",
		Env:       &exercises.Env{Embedded: k8slings.FS, Validator: schema.New(k8slings.Schemas)},
		Source:    k8slings.FS,
		Root:      ".",
		Solutions: k8slings.Solutions,
	})
	m.width, m.height = 100, 30
	settle(t, m, m.startRun())
	return m
}

// settle runs a command and feeds its message back to the model, until there
// is nothing left to run.
func settle(t *testing.T, m *Model, cmd tea.Cmd) {
	t.Helper()
	for range 5 {
		if cmd == nil {
			return
		}
		msg := cmd()
		if _, ok := msg.(tea.BatchMsg); ok {
			return
		}
		_, cmd = m.Update(msg)
	}
}

func press(t *testing.T, m *Model, key string) {
	t.Helper()
	var msg tea.KeyPressMsg
	switch key {
	case "esc":
		msg = tea.KeyPressMsg{Code: tea.KeyEscape}
	case "enter":
		msg = tea.KeyPressMsg{Code: tea.KeyEnter}
	case "down":
		msg = tea.KeyPressMsg{Code: tea.KeyDown}
	default:
		msg = tea.KeyPressMsg{Code: rune(key[0]), Text: key}
	}
	_, cmd := m.Update(msg)
	settle(t, m, cmd)
}

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// screen is what the learner sees, without the colour codes.
func screen(m *Model) string { return ansi.ReplaceAllString(m.View().Content, "") }

func want(t *testing.T, m *Model, parts ...string) {
	t.Helper()
	s := screen(m)
	for _, p := range parts {
		if !strings.Contains(s, p) {
			t.Fatalf("the screen should show %q:\n%s", p, s)
		}
	}
}

func solve(t *testing.T, name string, withMarker bool) {
	t.Helper()
	ex, err := exercises.Find(name, "info.toml")
	if err != nil {
		t.Fatal(err)
	}
	sol, err := k8slings.Solutions.ReadFile(ex.SolutionPath())
	if err != nil {
		t.Fatal(err)
	}
	content := string(sol)
	if withMarker {
		content = "# I AM NOT DONE\n" + content
	}
	if err := os.WriteFile(ex.File(), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestShowsTheFailureOfTheFirstExercise(t *testing.T) {
	m := newModel(t)
	want(t, m, "pods1", "is not solved yet", "contianers", "h hint")
}

func TestSavingAFixShowsSuccess(t *testing.T) {
	m := newModel(t)
	solve(t, "pods1", true)
	settle(t, m, m.startRun())
	want(t, m, "pods1 works!", "I AM NOT DONE")
}

func TestNextMarksDoneAndMovesOn(t *testing.T) {
	m := newModel(t)
	press(t, m, "n")
	want(t, m, "Get pods1 to work first")

	solve(t, "pods1", true)
	settle(t, m, m.startRun())
	press(t, m, "n")
	want(t, m, "pods2", "pods1 is done.")
	if ex, _ := exercises.Find("pods1", "info.toml"); ex.State() != exercises.Done {
		t.Fatal("n should have removed the marker of pods1")
	}
}

func TestHintAndEscape(t *testing.T) {
	m := newModel(t)
	press(t, m, "h")
	want(t, m, "Hint for pods1", "Docs: https://kubernetes.io")
	press(t, m, "esc")
	want(t, m, "is not solved yet")
}

func TestSolutionAsksFirst(t *testing.T) {
	m := newModel(t)
	press(t, m, "s")
	want(t, m, "Show the solution of pods1?")
	press(t, m, "x")
	want(t, m, "is not solved yet")

	press(t, m, "s")
	press(t, m, "y")
	want(t, m, "Solution for pods1", "containers:")
}

func TestResetRestoresTheOriginal(t *testing.T) {
	m := newModel(t)
	solve(t, "pods1", true)
	settle(t, m, m.startRun())
	press(t, m, "r")
	want(t, m, "Reset pods1 to its original state?")
	press(t, m, "y")
	want(t, m, "Reset exercises/pods/pods1/exercise.yaml", "is not solved yet")
}

func TestCheckAll(t *testing.T) {
	m := newModel(t)
	solve(t, "pods2", false) // done and working
	press(t, m, "c")
	want(t, m, "Check all:", "✔ pods2", "✘ pods1")
	press(t, m, "esc")
	want(t, m, "pods1")
}

func TestListJumpsToAnExercise(t *testing.T) {
	m := newModel(t)
	press(t, m, "l")
	want(t, m, "pods1", "labels1")
	press(t, m, "down")
	press(t, m, "enter")
	want(t, m, "pods2")
}

func TestFullSaysNothingWasShortened(t *testing.T) {
	m := newModel(t)
	press(t, m, "f")
	want(t, m, "Nothing was shortened")
}

func TestQuit(t *testing.T) {
	m := newModel(t)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Fatal("q should quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("q should produce a QuitMsg")
	}
}
