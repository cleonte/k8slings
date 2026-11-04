package cmd

import (
	"context"

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
	})
}
