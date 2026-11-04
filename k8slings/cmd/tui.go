package cmd

import (
	"context"
	"errors"
)

// runTUI is replaced by the full-screen view in the next commit.
func runTUI(ctx context.Context, changes <-chan []string) error {
	return errors.New("the full-screen view is not available yet; use --plain")
}
