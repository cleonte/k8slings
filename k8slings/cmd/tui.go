package cmd

import (
	"context"
	"os"
	"strings"

	"github.com/cleonte/k8slings"
	"github.com/cleonte/k8slings/k8slings/tui"
)

func runTUI(ctx context.Context, changes <-chan []string) error {
	return tui.Run(ctx, tui.Config{
		InfoFile:  infoFile,
		Changes:   changes,
		Env:       newEnv(),
		Source:    k8slings.FS,
		Root:      ".",
		Solutions: k8slings.Solutions,
		Editor:    editorCommand(),
	})
}

// editorCommand is the editor the learner has set up, for the e key of the
// full-screen view.
func editorCommand() string {
	if editor := strings.TrimSpace(os.Getenv("VISUAL")); editor != "" {
		return editor
	}
	return strings.TrimSpace(os.Getenv("EDITOR"))
}
