package cmd

import (
	"errors"
	"io/fs"

	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/cleonte/k8slings/k8slings/workspace"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func ResetCmd(src fs.FS, root string) *cobra.Command {
	return &cobra.Command{
		Use:   "reset <exercise name>",
		Short: "Restore an exercise to its original, broken state",
		Long: `Overwrites the exercise file with the original from the binary, which also
brings back the 'I AM NOT DONE' line. Your changes to that file are lost.`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := workspace.Reset(src, root, args[0])
			if errors.Is(err, exercises.ErrExerciseNotFound) {
				color.White("No exercise found for '%s'", args[0])
				return err
			}
			if err != nil {
				return err
			}
			color.Green("Restored %s", p)
			return nil
		},
	}
}
