package cmd

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/cleonte/k8slings/k8slings/watcher"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// debounce is how long the watcher waits for a burst of file events to settle.
const debounce = 150 * time.Millisecond

func WatchCmd(infoFile string) *cobra.Command {
	var plain bool

	watchCmd := &cobra.Command{
		Use:   "watch",
		Short: "Check exercises when files are saved",
		Long: `Checks the next exercise whenever you save a file.

In a terminal this opens a full-screen view. Use --plain for simple line-by-line
output; that is also what you get when input or output is not a terminal, for
example when piping.

In plain mode, type a command and press Enter: h (hint), l (list), r (run
again) or q (quit).`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			root := filepath.Join(filepath.Dir(infoFile), "exercises")
			changes, err := watcher.Watch(ctx, root, debounce)
			if err != nil {
				color.Red("Can't watch %s: %s", root, err)
				color.Yellow("Run k8slings from the folder that holds the exercises, or create one with `k8slings init`.")
				return err
			}

			if !plain && isTerminal(cmd.InOrStdin()) && isTerminal(cmd.OutOrStdout()) {
				return runTUI(ctx, changes)
			}

			watchLoop(cmd.InOrStdin(), changes, watchActions{
				run:  func() { _ = runNext(infoFile) },
				list: func() { _ = ListCmd(infoFile).RunE(nil, nil) },
				hint: func() { printNextHint(infoFile) },
			})
			return nil
		},
	}
	watchCmd.Flags().BoolVar(&plain, "plain", false, "print plain lines instead of the full-screen view")
	return watchCmd
}

// isTerminal reports whether f is an interactive terminal (and not, say, a pipe).
func isTerminal(f any) bool {
	file, ok := f.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

func printNextHint(infoFile string) {
	ex, err := exercises.NextPending(infoFile)
	if err != nil {
		color.Yellow("There is no pending exercise to give a hint for.")
		return
	}
	color.Cyan(ex.Hint)
	if ex.Docs != "" {
		color.White("\nDocs: %s", ex.Docs)
	}
}

type watchActions struct {
	run, list, hint func()
}

// watchLoop runs the next exercise now and whenever files change, and answers
// the commands typed by the learner, until they quit or input ends. Everything
// happens in this one loop, so a run never overlaps with another run or with a
// command.
func watchLoop(in io.Reader, changes <-chan []string, actions watchActions) {
	actions.run()

	lines := make(chan string)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(in)
		for scanner.Scan() {
			lines <- strings.TrimSpace(scanner.Text())
		}
	}()

	for {
		select {
		case _, ok := <-changes:
			if !ok {
				return
			}
			actions.run()

		case line, ok := <-lines:
			if !ok {
				return
			}
			switch line {
			case "l", "list":
				actions.list()
			case "h", "hint":
				actions.hint()
			case "r", "run":
				actions.run()
			case "q", "quit", "exit":
				color.Green("Bye from k8slings o/")
				return
			default:
				color.Yellow("commands: h (hint), l (list), r (run again), q (quit)")
			}
		}
	}
}
