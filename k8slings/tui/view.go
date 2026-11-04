package tui

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/cleonte/k8slings/k8slings/exercises"
)

func (m *Model) View() tea.View {
	view := tea.NewView(m.header() + m.body())
	view.AltScreen = true
	view.WindowTitle = "k8slings"
	return view
}

func (m *Model) header() string {
	title := titleStyle.Render("k8slings")
	if m.total > 0 {
		title += fmt.Sprintf("  %d/%d (%.0f%%)", m.done, m.total, 100*float64(m.done)/float64(m.total))
	}
	if m.topic != "" {
		title += "  " + dimStyle.Render(m.topic)
	}
	if m.skipped > 0 {
		title += "  " + warnStyle.Render(fmt.Sprintf("%d skipped", m.skipped))
	}

	current := ""
	if m.status == statusRunning || m.status == statusFinished {
		current = fmt.Sprintf("%s  %s", m.exercise.Name, dimStyle.Render(m.exercise.File()))
	}
	rule := dimStyle.Render(strings.Repeat("─", max(m.width, 1)))
	return title + "\n" + current + "\n" + rule + "\n"
}

// available is the number of lines the body can use.
func (m *Model) available() int {
	return max(m.height-headerLines-footerLines, 1)
}

// maxScroll is how far the body can be scrolled before running out of lines.
func (m *Model) maxScroll() int {
	return max(len(m.lines())-m.available(), 0)
}

// lines is everything the body would show, before cutting it to the screen.
func (m *Model) lines() []string {
	switch m.mode {
	case modeCheck:
		return m.checkLines()
	case modeList:
		return m.listLines()
	case modeHint:
		return m.hintLines()
	case modeSolution:
		return m.solutionLines()
	case modeConfirmSolution:
		return []string{
			warnStyle.Render("Show the solution of " + m.exercise.Name + "?"),
			"It spoils the exercise: try a bit longer, or read the hint (esc, then h) first.",
			"",
			"y to show it, any other key to go back",
		}
	case modeConfirmReset:
		return []string{
			warnStyle.Render("Reset " + m.exercise.Name + " to its original state?"),
			"This discards your changes to " + m.exercise.File() + ".",
			"",
			"y to reset it, any other key to keep your work",
		}
	}

	switch m.status {
	case statusStarting:
		return []string{"Starting…"}
	case statusRunning:
		return []string{dimStyle.Render("Checking " + m.exercise.Name + "…")}
	case statusAllDone:
		return []string{
			okStyle.Render("Congratulations! You have completed all the exercises that can run here."),
			"",
			"If you enjoyed it, please give the repository a star:",
			"https://github.com/cleonte/k8slings",
		}
	case statusNothingToRun:
		return []string{warnStyle.Render("There are no exercises to run."), "Run k8slings from the folder that holds info.toml, or create one with `k8slings init`."}
	case statusError:
		return []string{errStyle.Render(m.message), "Run k8slings from the folder that holds info.toml, or create one with `k8slings init`."}
	case statusFinished:
		return m.resultLines()
	}
	return nil
}

func (m *Model) hintLines() []string {
	lines := []string{titleStyle.Render("Hint for " + m.exercise.Name), ""}
	hint := strings.TrimSpace(m.exercise.Hint)
	if hint == "" {
		hint = "There is no hint for this exercise."
	}
	lines = append(lines, strings.Split(hint, "\n")...)
	if m.exercise.Docs != "" {
		lines = append(lines, "", dimStyle.Render("Docs: "+m.exercise.Docs))
	}
	return lines
}

func (m *Model) solutionLines() []string {
	lines := []string{titleStyle.Render("Solution for " + m.exercise.Name), ""}
	source, err := fs.ReadFile(m.cfg.Solutions, m.exercise.SolutionPath())
	if err != nil {
		return append(lines, errStyle.Render(err.Error()))
	}
	return append(lines, strings.Split(strings.TrimRight(string(source), "\n"), "\n")...)
}

func (m *Model) resultLines() []string {
	var failed *exercises.FailedError
	switch {
	case exercises.IsSkipped(m.runErr):
		return []string{warnStyle.Render(m.runErr.Error())}

	case errors.As(m.runErr, &failed):
		lines := []string{errStyle.Render("✘ " + m.exercise.Name + " is not solved yet:"), ""}
		for _, l := range failed.Lines {
			lines = append(lines, errStyle.Render("  • ")+l)
		}
		return append(lines, "", dimStyle.Render("Edit "+m.exercise.File()+" and save. Press h for a hint."))

	case m.runErr != nil:
		return []string{errStyle.Render(m.runErr.Error())}
	}

	lines := []string{okStyle.Render("✔ " + m.exercise.Name + " works!"), ""}
	if m.exercise.State() == exercises.Pending {
		lines = append(lines, warnStyle.Render("Remove the `# I AM NOT DONE` line from the file, or press n, to move on to the next exercise."))
	}
	return lines
}

func (m *Model) body() string {
	lines := m.lines()
	available := m.available()

	start := min(m.scroll, max(len(lines)-available, 0))
	lines = lines[start:]
	if len(lines) > available {
		hidden := len(lines) - (available - 1)
		lines = append(lines[:available-1], dimStyle.Render(fmt.Sprintf("… %d more lines (↓ to scroll)", hidden)))
	}
	for len(lines) < available {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n") + "\n" + m.footer()
}

func (m *Model) footer() string {
	switch {
	case m.notice != "":
		return warnStyle.Render(m.notice)
	case m.mode == modeConfirmSolution || m.mode == modeConfirmReset:
		return dimStyle.Render("y yes   any other key: no   q quit")
	case m.mode == modeList && m.list.SettingFilter():
		return dimStyle.Render("enter apply filter   esc cancel")
	case m.mode == modeList:
		return dimStyle.Render("enter open   / filter   ↑↓ move   esc back   q quit")
	case m.mode != modeResult:
		return dimStyle.Render("esc back   ↑↓ scroll   q quit")
	case m.hasExercise():
		keys := []string{"h hint", "s solution", "r reset", "n next", "l list", "c check all", "f full", "q quit"}
		withScroll := slices.Insert(slices.Clone(keys), len(keys)-1, "↑↓ scroll")
		line := strings.Join(withScroll, "   ")
		if lipgloss.Width(line) > m.width {
			line = strings.Join(keys, "   ")
		}
		return dimStyle.Render(line)
	}
	return dimStyle.Render("q quit")
}
