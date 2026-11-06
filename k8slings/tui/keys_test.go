package tui

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	tea "charm.land/bubbletea/v2"
	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/cleonte/k8slings/k8slings/schema"
)

func key(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code, Text: string(code)} }

func typeText(t *testing.T, m *Model, s string) {
	t.Helper()
	for _, r := range s {
		_, cmd := m.Update(key(r))
		settle(t, m, cmd)
	}
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

// miniModel is a model over a workspace made of the files given, with one
// exercise "a" that is checked against `check` (schema off).
func miniModel(t *testing.T, info string, files map[string]string, check string) *Model {
	t.Helper()
	t.Chdir(t.TempDir())
	if info != "" {
		if err := os.WriteFile("info.toml", []byte(info), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for name, content := range files {
		if err := os.MkdirAll(name[:strings.LastIndex(name, "/")], 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	embedded := fstest.MapFS{"exercises/t/a/check.yaml": {Data: []byte(check)}}
	m := New(Config{
		InfoFile: "info.toml",
		Env:      &exercises.Env{Embedded: embedded, Validator: schema.New(embedded)},
	})
	m.width, m.height = 100, 30
	settle(t, m, m.startRun())
	return m
}

const oneExercise = "[[exercises]]\nname='a'\npath='exercises/t/a'\ntopic='t'\nmode='manifest'\nhint='h'\n"
const alwaysTrue = "schema: \"off\"\nobjects:\n  - select: {kind: Pod}\n    assert:\n      - {expr: \"true\", message: \"never\"}\n"

func TestAllDoneScreen(t *testing.T) {
	m := miniModel(t, oneExercise, map[string]string{"exercises/t/a/exercise.yaml": "kind: Pod\n"}, alwaysTrue)
	want(t, m, "Congratulations", "k8slings")
	if m.status != statusAllDone {
		t.Fatalf("status = %d", m.status)
	}
}

func TestNothingToRunScreen(t *testing.T) {
	m := miniModel(t, "", nil, alwaysTrue)
	want(t, m, "Failed to read the exercises") // no info.toml at all

	m = miniModel(t, "exercises = []\n", nil, alwaysTrue)
	want(t, m, "There are no exercises to run")
}

func TestErrorScreenSaysWhatToDo(t *testing.T) {
	m := miniModel(t, "", nil, alwaysTrue)
	want(t, m, "Failed to read the exercises", "k8slings init")
	// The keys that need an exercise do nothing, instead of crashing.
	for _, k := range []string{"h", "s", "r", "n", "f"} {
		press(t, m, k)
	}
	want(t, m, "Failed to read the exercises")
}

func TestQIsAQuitKeyButALetterInTheFilter(t *testing.T) {
	m := newModel(t)
	press(t, m, "l")
	want(t, m, "labels1")

	// "/" starts a filter: from now on q is a letter to search for.
	typeText(t, m, "/")
	_, cmd := m.Update(key('q'))
	if isQuit(cmd) {
		t.Fatal("q must not quit while a filter is being typed")
	}
	if m.mode != modeList {
		t.Fatal("the list should still be open")
	}
	// esc cancels the filter, esc again leaves the list, q then quits.
	press(t, m, "esc")
	press(t, m, "esc")
	if m.mode != modeResult {
		t.Fatalf("mode = %d, want the result screen", m.mode)
	}
	_, cmd = m.Update(key('q'))
	if !isQuit(cmd) {
		t.Fatal("q should quit outside of a filter")
	}
}

func TestQuitFromTheListAndWithCtrlC(t *testing.T) {
	m := newModel(t)
	press(t, m, "l")
	_, cmd := m.Update(key('q'))
	if !isQuit(cmd) {
		t.Fatal("q should quit from the list")
	}

	m = newModel(t)
	_, cmd = m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if !isQuit(cmd) {
		t.Fatal("ctrl+c should quit")
	}
}

func TestFilterNarrowsTheListByNameAndTopic(t *testing.T) {
	m := newModel(t)
	press(t, m, "l")
	typeText(t, m, "/rbac")
	press(t, m, "enter") // applies the filter
	// The list itself, not the whole screen: the header always names the
	// current exercise.
	list := ansi.ReplaceAllString(strings.Join(m.listLines(), "\n"), "")
	if !strings.Contains(list, "rbac1") || !strings.Contains(list, "rbac2") {
		t.Fatalf("the filter should keep the rbac exercises:\n%s", list)
	}
	if strings.Contains(list, "pods1") || strings.Contains(list, "labels1") {
		t.Fatalf("the filter should hide the others:\n%s", list)
	}
}

func TestScrollingStaysInsideTheContent(t *testing.T) {
	m := newModel(t)
	m.height = 10 // a few lines only
	press(t, m, "c")
	if m.maxScroll() == 0 {
		t.Fatal("the check results should be longer than the screen")
	}

	for range 3 {
		press(t, m, "down")
	}
	if m.scroll != 3 {
		t.Fatalf("scroll = %d after three downs", m.scroll)
	}
	press(t, m, "up")
	if m.scroll != 2 {
		t.Fatalf("scroll = %d after an up", m.scroll)
	}
	for range 200 {
		press(t, m, "down")
	}
	if m.scroll != m.maxScroll() {
		t.Fatalf("scroll = %d, must stop at %d", m.scroll, m.maxScroll())
	}
	for range 200 {
		press(t, m, "up")
	}
	if m.scroll != 0 {
		t.Fatalf("scroll = %d, must stop at 0", m.scroll)
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	settle(t, m, cmd)
	if m.scroll != min(m.available(), m.maxScroll()) {
		t.Fatalf("a page down scrolled to %d", m.scroll)
	}
	press(t, m, "esc") // leaving check-all runs the exercise again, from the top
	if m.scroll != 0 {
		t.Fatalf("scroll = %d after leaving", m.scroll)
	}
}

func TestResizeChangesTheLayout(t *testing.T) {
	m := newModel(t)
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 12})
	if m.width != 60 || m.height != 12 {
		t.Fatalf("size = %dx%d", m.width, m.height)
	}
	if !strings.Contains(screen(m), strings.Repeat("─", 60)) || strings.Contains(screen(m), strings.Repeat("─", 61)) {
		t.Fatalf("the rule under the header should be 60 wide:\n%s", screen(m))
	}
	if got := len(strings.Split(m.body(), "\n")); got > 12 {
		t.Fatalf("the body has %d lines on a 12 line screen", got)
	}
}

func TestFooterOnlyOffersWhatWorks(t *testing.T) {
	m := newModel(t)
	s := screen(m)
	for _, k := range []string{"h hint", "s solution", "r reset", "n next", "l list", "c check all", "q quit"} {
		if !strings.Contains(s, k) {
			t.Fatalf("the footer should offer %q:\n%s", k, s)
		}
	}
	if strings.Contains(s, "f full") {
		t.Fatal("f full must not be offered: nothing is ever shortened, so it would show nothing")
	}
	// Narrow screens drop the scroll hint rather than overflow.
	m.width = 55
	if lines := strings.Split(m.footer(), "\n"); len(lines) != 1 {
		t.Fatalf("the footer wrapped: %q", m.footer())
	}
}

func TestHeaderPointsToDoctorWhenExercisesAreSkipped(t *testing.T) {
	m := newModel(t) // no cluster: the cluster exercises are skipped
	want(t, m, "skipped (see k8slings doctor)")
	if m.skipped == 0 {
		t.Fatal("cluster exercises should be skipped without a cluster")
	}
}
