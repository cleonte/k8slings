package tui

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"
	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/cleonte/lingscore/tui/checkall"
)

// checkWorkers is how many exercises are checked at the same time while
// checking all of them. Cluster exercises each work in a namespace of their
// own, so they can run side by side; more than this would only crowd a small
// cluster.
const checkWorkers = 4

// startCheck checks every exercise, a few at a time, without changing any file.
// The state of each one is shown as it goes, and esc cancels what is left.
func (m *Model) startCheck() tea.Cmd {
	if m.check.Active() {
		m.notice = "Already checking the exercises."
		return nil
	}
	all, err := exercises.List(m.cfg.InfoFile)
	if err != nil || len(all) == 0 {
		m.notice = "There are no exercises to check."
		return nil
	}

	m.stopRun()
	m.mode, m.scroll = modeCheck, 0

	// Checking every exercise must not break the cluster for fix exercises that
	// have not been started.
	run := m.cfg.Run
	return m.check.Start(all, checkWorkers, func(ctx context.Context, e exercises.Exercise) checkall.Result {
		return checkResult(run(exercises.NoSeed(ctx), e))
	})
}

// checkResult turns what running an exercise said into what the view shows.
func checkResult(_ exercises.Result, err error) checkall.Result {
	var failed *exercises.FailedError
	switch {
	case err == nil:
		return checkall.Result{Status: checkall.Passed}
	case exercises.IsSkipped(err):
		return checkall.Result{Status: checkall.Skipped, Detail: err.Error()}
	case errors.As(err, &failed) && len(failed.Lines) > 0:
		return checkall.Result{Status: checkall.Failed, Detail: failed.Lines[0]}
	default:
		return checkall.Result{Status: checkall.Failed, Detail: err.Error()}
	}
}

func (m *Model) checkLines() []string {
	lines := []string{titleStyle.Render(m.check.Title()), ""}
	for _, item := range m.check.Items {
		lines = append(lines, checkLine(item))
	}
	return lines
}

func checkLine(item checkall.Item[exercises.Exercise]) string {
	name := item.Value.Name
	switch item.Status {
	case checkall.Waiting:
		return dimStyle.Render("· " + name)
	case checkall.Running:
		return dimStyle.Render("… " + name)
	case checkall.Skipped:
		return warnStyle.Render("• "+name) + dimStyle.Render("  "+item.Detail)
	case checkall.Passed:
		if item.Value.State() == exercises.Pending {
			return okStyle.Render("✔ "+name) + dimStyle.Render("  works, the I AM NOT DONE marker is still there")
		}
		return okStyle.Render("✔ " + name)
	}

	// A failure: say what is wrong, and shout if the exercise was finished before.
	if item.Value.State() == exercises.Done {
		return warnStyle.Render("✗ "+name+"  was finished, but now fails: ") + dimStyle.Render(item.Detail)
	}
	return errStyle.Render("✗ "+name) + dimStyle.Render("  "+item.Detail)
}
