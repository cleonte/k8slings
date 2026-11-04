package cmd

import (
	"errors"
	"fmt"

	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func RunCmd(infoFile string) *cobra.Command {
	return &cobra.Command{
		Use:   "run next | <exercise name>",
		Short: "Run a single exercise",
		Long: `example next pending exercise : k8slings run next
example specific exercise : k8slings run pods1`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var ex exercises.Exercise
			var err error
			if args[0] == "next" {
				ex, err = exercises.NextPending(infoFile)
			} else {
				ex, err = exercises.Find(args[0], infoFile)
			}
			if errors.Is(err, exercises.ErrExerciseNotFound) {
				color.White("No exercise found for '%s'", args[0])
				return err
			}
			if errors.Is(err, exercises.ErrNoPending) {
				color.Green("🎉 Every exercise is done.")
				return nil
			}
			if err != nil {
				return err
			}

			result, err := ex.Run(newEnv())
			if exercises.IsSkipped(err) {
				color.Yellow(err.Error())
				return err
			}
			if err != nil {
				printFailure(ex, err)
				return err
			}
			color.Green("✅ %s passes (%d objects checked).\n", result.Exercise.Name, result.Objects)
			if ex.State() == exercises.Pending {
				color.White("Remove the 'I AM NOT DONE' line from %s to keep going.", ex.File())
				return fmt.Errorf("exercise is still pending")
			}
			return nil
		},
	}
}
