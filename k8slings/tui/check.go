package tui

import (
	"context"
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/cleonte/k8slings/k8slings/exercises"
)

type checkStatus int

const (
	checkPassed checkStatus = iota
	checkFailed
	checkSkipped
)

type checkItem struct {
	exercise exercises.Exercise
	status   checkStatus
	detail   string // first problem, for a failure
	broken   bool   // done (no marker) but failing
}

// startCheck runs every exercise without changing any file.
func (m *Model) startCheck() tea.Cmd {
	all, err := exercises.List(m.cfg.InfoFile)
	if err != nil || len(all) == 0 {
		m.notice = "There are no exercises to check."
		return nil
	}
	m.stopRun()
	m.mode, m.scroll = modeCheck, 0
	m.checkItems = nil

	run := m.cfg.Run
	return func() tea.Msg {
		items := make([]checkItem, 0, len(all))
		for _, e := range all {
			it := checkItem{exercise: e}
			_, err := run(context.Background(), e)
			var failed *exercises.FailedError
			switch {
			case exercises.IsSkipped(err):
				it.status, it.detail = checkSkipped, err.Error()
			case errors.As(err, &failed):
				it.status, it.detail = checkFailed, failed.Lines[0]
				it.broken = e.State() == exercises.Done
			case err != nil:
				it.status, it.detail = checkFailed, err.Error()
			}
			items = append(items, it)
		}
		return checkDoneMsg{items: items}
	}
}

func (m *Model) checkLines() []string {
	if m.checkItems == nil {
		return []string{dimStyle.Render("Checking every exercise…")}
	}
	var lines []string
	var passed, failed, skipped, broken int
	for _, it := range m.checkItems {
		switch it.status {
		case checkPassed:
			passed++
			lines = append(lines, okStyle.Render("✔ ")+it.exercise.Name)
		case checkSkipped:
			skipped++
			lines = append(lines, warnStyle.Render("↷ "+it.exercise.Name)+dimStyle.Render("  "+it.detail))
		default:
			failed++
			line := errStyle.Render("✘ ") + it.exercise.Name + dimStyle.Render("  "+it.detail)
			if it.broken {
				broken++
				line += warnStyle.Render("  (was done, now broken)")
			}
			lines = append(lines, line)
		}
	}
	summary := fmt.Sprintf("%d pass, %d fail, %d skipped", passed, failed, skipped)
	if broken > 0 {
		summary += fmt.Sprintf(", %d finished exercises broke", broken)
	}
	return append([]string{titleStyle.Render("Check all: " + summary), ""}, lines...)
}
