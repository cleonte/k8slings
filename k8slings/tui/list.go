package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/cleonte/lingscore/tui/jumplist"
)

// openList shows every exercise, with the current one selected.
func (m *Model) openList() {
	all, err := exercises.List(m.cfg.InfoFile)
	if err != nil || len(all) == 0 {
		m.notice = "There are no exercises to list."
		return
	}
	av := m.cfg.Env.Availability(context.Background())
	items := make([]jumplist.Item, len(all))
	current := 0
	for i, e := range all {
		reason := e.SkipReason(av)
		items[i] = jumplist.Item{Name: e.Name, Topic: e.Topic, Done: e.State() == exercises.Done, Skipped: reason != "", Reason: reason}
		if m.hasExercise() && e.Name == m.exercise.Name {
			current = i
		}
	}
	m.list = jumplist.New(items, current, m.width, m.available())
	m.mode, m.scroll = modeList, 0
}

// listKey handles a key while the list is open.
func (m *Model) listKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.notice = ""
	action, cmd := m.list.Key(msg)
	switch action {
	case jumplist.Quit:
		m.stopRun()
		m.check.Stop()
		return m, tea.Quit
	case jumplist.Back:
		m.mode = modeResult
	case jumplist.Open:
		it, _ := m.list.Selected()
		m.pinned = it.Name
		return m, m.startRun()
	}
	return m, cmd
}

func (m *Model) listLines() []string { return m.list.Lines() }
