// Package tui is the full-screen terminal interface of `k8slings watch`: the
// progress at the top, the result of the current exercise below, refreshed
// every time a file is saved.
package tui

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/cleonte/k8slings/k8slings/workspace"
)

// RunFunc checks an exercise.
type RunFunc func(ctx context.Context, ex exercises.Exercise) (exercises.Result, error)

type Config struct {
	InfoFile string
	// Changes delivers the files that were saved, see the watcher package.
	Changes <-chan []string
	// Env is what checking an exercise needs.
	Env *exercises.Env
	// Run defaults to checking the exercise with Env.
	Run RunFunc
	// Source holds the original exercises, used to reset one. Root is the folder
	// the exercise paths of the info file are relative to. Without a Source the
	// reset key says it is not available.
	Source fs.FS
	Root   string
	// Solutions holds the solutions shown on request, laid out like
	// solutions/. Without it the solution key says it is not available.
	Solutions fs.FS
}

type status int

const (
	statusStarting status = iota
	statusRunning
	statusFinished
	statusAllDone
	statusNothingToRun
	statusError
)

// mode is what the lower part of the screen is showing.
type mode int

const (
	modeResult mode = iota // the result of the run
	modeHint
	modeSolution
	modeConfirmSolution
	modeConfirmReset
	modeCheck // the result of checking every exercise
	modeList  // every exercise, to jump to one
)

type Model struct {
	cfg    Config
	width  int
	height int

	mode   mode
	scroll int    // lines scrolled off the top of the body
	notice string // one line shown instead of the key help

	status   status
	message  string // for statusError
	done     int
	total    int
	skipped  int
	topic    string // e.g. "Topic: pods (2/5)"
	exercise exercises.Exercise
	avail    exercises.Availability // what this machine can run, looked at on every run
	// pinned is the exercise picked from the list. It is the one that runs
	// until n moves on, instead of the first pending one.
	pinned string
	list   list.Model

	// Checking every exercise (the c key). Like runs, the results of a check
	// that was cancelled carry an older id and are ignored.
	checkID     int
	checkCtx    context.Context
	checkCancel context.CancelFunc
	checkItems  []checkItem
	checkNext   int // index of the next exercise to start
	checkActive int // how many are running now

	// The run in progress. Results of earlier runs carry an older id and are
	// ignored: a save during a run starts a new one.
	runID  int
	cancel context.CancelFunc
	result exercises.Result
	runErr error
}

type startMsg struct{}

type changesMsg struct{ files []string }

type changesClosedMsg struct{}

type runDoneMsg struct {
	id     int
	result exercises.Result
	err    error
}

type checkDoneMsg struct {
	id    int
	index int
	err   error
}

func New(cfg Config) *Model {
	if cfg.Run == nil {
		cfg.Run = func(ctx context.Context, ex exercises.Exercise) (exercises.Result, error) {
			return ex.Run(ctx, cfg.Env)
		}
	}
	return &Model{cfg: cfg, width: 80, height: 24}
}

// Run shows the interface until the learner quits or ctx ends.
func Run(ctx context.Context, cfg Config) error {
	_, err := tea.NewProgram(New(cfg), tea.WithContext(ctx)).Run()
	if errors.Is(err, tea.ErrProgramKilled) && ctx.Err() != nil {
		return nil
	}
	return err
}

func (m *Model) Init() tea.Cmd {
	return func() tea.Msg { return startMsg{} }
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.mode == modeList {
			m.list.SetSize(m.width, m.available())
		}

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case startMsg:
		return m, tea.Batch(m.startRun(), m.waitForChanges())

	case changesMsg:
		if m.checking() {
			// A save should not throw away a check that is half way; leaving the
			// check with esc runs the exercise again.
			return m, m.waitForChanges()
		}
		return m, tea.Batch(m.startRun(), m.waitForChanges())

	case changesClosedMsg:
		// The watcher stopped; whatever is on screen stays, and q still works.

	case checkDoneMsg:
		return m, m.finishCheck(msg)

	case runDoneMsg:
		if msg.id == m.runID {
			m.status, m.result, m.runErr = statusFinished, msg.result, msg.err
		}

	default:
		// What the list asks for itself, e.g. the matches of a filter.
		if m.mode == modeList {
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.mode == modeList {
		return m.listKey(msg)
	}
	key := msg.String()
	if key == "q" || key == "ctrl+c" {
		m.stopRun()
		m.stopCheck()
		return m, tea.Quit
	}
	m.notice = ""

	// A question is open: only yes or no answer it.
	if m.mode == modeConfirmSolution || m.mode == modeConfirmReset {
		pending := m.mode
		m.mode, m.scroll = modeResult, 0
		if key != "y" {
			return m, nil
		}
		if pending == modeConfirmSolution {
			m.showSolution()
			return m, nil
		}
		return m, m.reset()
	}

	// While checking, only leaving, scrolling and a second c (which says it is
	// already going) make sense.
	if m.mode == modeCheck && !slices.Contains([]string{"esc", "c", "down", "j", "up", "k", "pgdown", "space", "pgup"}, key) {
		return m, nil
	}

	switch key {
	case "esc":
		if m.mode == modeCheck {
			return m, m.startRun()
		}
		m.mode, m.scroll = modeResult, 0
	case "h":
		m.toggle(modeHint)
	case "s":
		m.askToShowSolution()
	case "r":
		m.askToReset()
	case "n":
		return m, m.next()
	case "c":
		return m, m.startCheck()
	case "l":
		m.openList()
	case "f":
		m.showFull()
	case "down", "j":
		m.scroll = min(m.scroll+1, m.maxScroll())
	case "up", "k":
		m.scroll = max(m.scroll-1, 0)
	case "pgdown", "space":
		m.scroll = min(m.scroll+m.available(), m.maxScroll())
	case "pgup":
		m.scroll = max(m.scroll-m.available(), 0)
	}
	return m, nil
}

// hasExercise reports whether there is a current exercise to act on.
func (m *Model) hasExercise() bool {
	return m.status == statusRunning || m.status == statusFinished
}

func (m *Model) toggle(target mode) {
	if !m.hasExercise() {
		return
	}
	m.scroll = 0
	if m.mode == target {
		m.mode = modeResult
		return
	}
	m.mode = target
}

// showFull would show the failure with nothing shortened. In manifest mode
// every problem is shown in full already, so it only says so; the key is kept
// so that cluster exercises, whose output can be long, can use it later.
func (m *Model) showFull() {
	if m.status == statusFinished && m.runErr != nil && !exercises.IsSkipped(m.runErr) {
		m.notice = "Nothing was shortened: this is all of the output."
	}
}

func (m *Model) askToShowSolution() {
	switch {
	case !m.hasExercise():
	case m.cfg.Solutions == nil:
		m.notice = "Solutions are not available here."
	default:
		m.mode, m.scroll = modeConfirmSolution, 0
	}
}

func (m *Model) showSolution() {
	if _, err := fs.Stat(m.cfg.Solutions, m.exercise.SolutionPath()); err != nil {
		m.notice = "No solution available for " + m.exercise.Name + " yet."
		return
	}
	m.mode, m.scroll = modeSolution, 0
}

func (m *Model) askToReset() {
	switch {
	case !m.hasExercise():
	case m.cfg.Source == nil:
		m.notice = "Reset is not available here."
	default:
		m.mode, m.scroll = modeConfirmReset, 0
	}
}

// reset puts the original exercise back and runs it again.
func (m *Model) reset() tea.Cmd {
	root := m.cfg.Root
	if root == "" {
		root = "."
	}
	path, err := workspace.Reset(m.cfg.Source, root, m.exercise.Name)
	if err != nil {
		m.notice = "Reset failed: " + err.Error()
		return nil
	}
	cmd := m.startRun()
	m.notice = "Reset " + path + " to its original state."
	return cmd
}

// next moves on from an exercise that works: it takes out the
// "I AM NOT DONE" marker, which is what the learner would do by hand, and runs
// the next one.
func (m *Model) next() tea.Cmd {
	switch {
	case !m.hasExercise():
		return nil
	case m.status == statusFinished && exercises.IsSkipped(m.runErr):
		// Only an exercise picked from the list can be a skipped one. It is not
		// done, so its marker stays; go back to the first pending exercise.
		name := m.exercise.Name
		m.pinned = ""
		cmd := m.startRun()
		m.notice = name + " is skipped here, so it is not done."
		return cmd
	case m.status != statusFinished || m.runErr != nil:
		m.notice = "Get " + m.exercise.Name + " to work first, then n moves on to the next exercise."
		return nil
	}

	if err := m.exercise.MarkDone(); err != nil && !errors.Is(err, exercises.ErrNotMarked) {
		m.notice = "Could not mark " + m.exercise.Name + " as done: " + err.Error()
		return nil
	}
	name := m.exercise.Name
	m.pinned = ""
	cmd := m.startRun()
	m.notice = name + " is done."
	return cmd
}

func (m *Model) stopRun() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.runID++
}

// startRun cancels the run in progress, works out which exercise is next and
// starts checking it.
func (m *Model) startRun() tea.Cmd {
	m.stopRun()
	m.stopCheck()
	m.mode, m.scroll = modeResult, 0

	done, total, err := exercises.Progress(m.cfg.InfoFile)
	if err != nil {
		m.status, m.message = statusError, fmt.Sprintf("Failed to read the exercises: %s", err)
		return nil
	}
	m.done, m.total = done, total
	m.avail = m.cfg.Env.Availability(context.Background())
	if skipped, err := exercises.Skipped(m.cfg.InfoFile, m.avail); err == nil {
		m.skipped = len(skipped)
	}

	exercise, err := m.nextExercise()
	switch {
	case total == 0:
		m.status = statusNothingToRun
		return nil
	case errors.Is(err, exercises.ErrNoPending):
		m.status = statusAllDone
		return nil
	case err != nil:
		m.status, m.message = statusError, "Failed to find the next exercise: "+err.Error()
		return nil
	}
	m.exercise = exercise
	m.topic = m.topicLine(exercise)

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.status, m.result, m.runErr = statusRunning, exercises.Result{}, nil

	id, run := m.runID, m.cfg.Run
	return func() tea.Msg {
		result, err := run(ctx, exercise)
		return runDoneMsg{id: id, result: result, err: err}
	}
}

// nextExercise is the one picked from the list, or else the first pending one.
func (m *Model) nextExercise() (exercises.Exercise, error) {
	if m.pinned != "" {
		if exercise, err := exercises.Find(m.pinned, m.cfg.InfoFile); err == nil {
			return exercise, nil
		}
		m.pinned = ""
	}
	return exercises.NextPending(m.cfg.InfoFile, m.avail)
}

func (m *Model) topicLine(exercise exercises.Exercise) string {
	topics, err := exercises.ProgressByTopic(m.cfg.InfoFile)
	if err != nil {
		return ""
	}
	for _, t := range topics {
		if t.Topic == exercise.Topic {
			return fmt.Sprintf("Topic: %s (%d/%d)", t.Topic, t.Done, t.Total)
		}
	}
	return ""
}

func (m *Model) waitForChanges() tea.Cmd {
	changes := m.cfg.Changes
	if changes == nil {
		return nil
	}
	return func() tea.Msg {
		files, ok := <-changes
		if !ok {
			return changesClosedMsg{}
		}
		return changesMsg{files: files}
	}
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	okStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	headerLines = 3 // title, exercise, rule
	footerLines = 1
)
