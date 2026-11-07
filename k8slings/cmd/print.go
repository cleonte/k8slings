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
	if ex.MarkerGone() && ex.State() == exercises.Pending {
		color.Yellow("\nThe 'I AM NOT DONE' line is gone, but %s has not passed yet, so it still counts as pending.", ex.Name)
	}
	if len(failed.Details) > 0 {
		fmt.Println()
		for _, line := range failed.Details {
			fmt.Println(line)
		}
	}
	if ex.Mode == exercises.ModeFix {
		color.White("\nRepair it with: k8slings kubectl -n %s ...", ex.Namespace())
		color.White("The ticket is in %s", ex.File())
	} else {
		color.White("\nEdit %s", ex.Location())
	}
	color.White("Need a nudge? k8slings hint %s", ex.Name)
}

// printSeeded tells the learner that the cluster was just broken on purpose.
func printSeeded(ex exercises.Exercise) {
	color.Yellow("🧰 %s: the cluster is now broken on purpose.\n", ex.Name)
	color.White("Read the ticket:  %s", ex.File())
	color.White("Look around:      k8slings kubectl -n %s get pods", ex.Namespace())
	color.White("Repair it, then run `k8slings run %s` again to check.", ex.Name)
	color.White("(Starting over: k8slings reset %s)", ex.Name)
}
