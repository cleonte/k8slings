package cmd

import (
	"errors"
	"fmt"

	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/fatih/color"
)

// printFailure explains why an exercise is not solved yet.
func printFailure(ex exercises.Exercise, err error) {
	var failed *exercises.FailedError
	if !errors.As(err, &failed) {
		color.Red("❌ %s: %s", ex.Name, err)
		return
	}
	color.Red("❌ %s is not solved yet:\n", ex.Name)
	for _, line := range failed.Lines {
		color.Red("  • %s", line)
	}
	if len(failed.Details) > 0 {
		fmt.Println()
		for _, line := range failed.Details {
			fmt.Println(line)
		}
	}
	color.White("\nEdit %s", ex.File())
	color.White("Need a nudge? k8slings hint %s", ex.Name)
}
