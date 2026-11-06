package tui

import (
	"context"
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/cleonte/k8slings/k8slings/exercises"
)

// checkWorkers is how many exercises are checked at the same time while
// checking all of them. Cluster exercises each work in a namespace of their
// own, so they can run side by side; more than this would only crowd a small
// cluster.
const checkWorkers = 4

type checkStatus int

const (
	checkWaiting checkStatus = iota
	checkRunning
	checkPassed
	checkFailed
	checkSkipped
)

type checkItem struct {
	exercise exercises.Exercise
	status   checkStatus
	detail   string // the first problem of a failure, or why it was skipped
}

// startCheck checks every exercise, a few at a time, without changing any file.
// The state of each one is shown as it goes, and esc cancels what is left.
func (m *Model) startCheck() tea.Cmd {
	if m.checking() {
		m.notice = "Already checking the exercises."
		return nil
	}
	all, err := exercises.List(m.cfg.InfoFile)
	if err != nil || len(all) == 0 {
		m.notice = "There are no exercises to check."
		return nil
	}

	m.stopRun()
	m.checkID++
	ctx, cancel := context.WithCancel(context.Background())
	m.checkCtx, m.checkCancel = ctx, cancel
	m.checkItems = make([]checkItem, len(all))
	for i, e := range all {
		m.checkItems[i] = checkItem{exercise: e}
	}
	m.checkNext, m.checkActive = 0, 0
	m.mode, m.scroll = modeCheck, 0

	var cmds []tea.Cmd
	for range checkWorkers {
		if cmd := m.startNextCheck(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

// startNextCheck starts the next exercise that has not been started yet.
func (m *Model) startNextCheck() tea.Cmd {
	if m.checkNext >= len(m.checkItems) {
		return nil
	}
	index := m.checkNext
	m.checkNext++
	m.checkItems[index].status = checkRunning
	m.checkActive++

	// Checking every exercise must not break the cluster for fix exercises that
	// have not been started.
	exercise, id, run, ctx := m.checkItems[index].exercise, m.checkID, m.cfg.Run, exercises.NoSeed(m.checkCtx)
	return func() tea.Msg {
		_, err := run(ctx, exercise)
		return checkDoneMsg{id: id, index: index, err: err}
	}
}

// finishCheck records the result of one exercise and starts the next.
func (m *Model) finishCheck(msg checkDoneMsg) tea.Cmd {
	// The result of a check that was cancelled carries an older id.
	if msg.id != m.checkID || m.checkCancel == nil || msg.index >= len(m.checkItems) {
		return nil
	}

	item := &m.checkItems[msg.index]
	var failed *exercises.FailedError
	switch {
	case msg.err == nil:
		item.status = checkPassed
	case exercises.IsSkipped(msg.err):
		item.status, item.detail = checkSkipped, msg.err.Error()
	case errors.As(msg.err, &failed) && len(failed.Lines) > 0:
		item.status, item.detail = checkFailed, failed.Lines[0]
	default:
		item.status, item.detail = checkFailed, msg.err.Error()
	}
	m.checkActive--

	if cmd := m.startNextCheck(); cmd != nil {
		return cmd
	}
	if m.checkActive == 0 {
		m.checkCancel()
		m.checkCancel = nil
	}
	return nil
}

// stopCheck cancels a check that is still going.
func (m *Model) stopCheck() {
	if m.checkCancel != nil {
		m.checkCancel()
		m.checkCancel = nil
		m.checkID++
	}
}

// checking reports whether exercises are still being checked.
func (m *Model) checking() bool { return m.checkCancel != nil }

func (m *Model) checkLines() []string {
	var passed, failed, skipped, done int
	for _, item := range m.checkItems {
		switch item.status {
		case checkPassed:
			passed++
		case checkFailed:
			failed++
		case checkSkipped:
			skipped++
		}
		if item.status >= checkPassed {
			done++
		}
	}

	title := fmt.Sprintf("Checked %d exercises: %d work, %d do not, %d skipped", len(m.checkItems), passed, failed, skipped)
	if m.checking() {
		title = fmt.Sprintf("Checking the exercises… %d/%d", done, len(m.checkItems))
	}
	lines := []string{titleStyle.Render(title), ""}
	for _, item := range m.checkItems {
		lines = append(lines, checkLine(item))
	}
	return lines
}

func checkLine(item checkItem) string {
	name := item.exercise.Name
	switch item.status {
	case checkWaiting:
		return dimStyle.Render("· " + name)
	case checkRunning:
		return dimStyle.Render("… " + name)
	case checkSkipped:
		return warnStyle.Render("• "+name) + dimStyle.Render("  "+item.detail)
	case checkPassed:
		if item.exercise.State() == exercises.Pending {
			return okStyle.Render("✔ "+name) + dimStyle.Render("  works, the I AM NOT DONE marker is still there")
		}
		return okStyle.Render("✔ " + name)
	}

	// A failure: say what is wrong, and shout if the exercise was finished before.
	if item.exercise.State() == exercises.Done {
		return warnStyle.Render("✗ "+name+"  was finished, but now fails: ") + dimStyle.Render(item.detail)
	}
	return errStyle.Render("✗ "+name) + dimStyle.Render("  "+item.detail)
}
