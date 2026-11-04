package cmd

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func SolutionCmd(infoFile string, solutions fs.FS) *cobra.Command {
	var reveal bool
	c := &cobra.Command{
		Use:   "solution <exercise name>",
		Short: "Show the solution of an exercise",
		Long: `Prints a working version of the exercise. It is hidden while the exercise is
pending, so try first; use --reveal to look anyway.`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ex, err := exercises.Find(args[0], infoFile)
			if errors.Is(err, exercises.ErrExerciseNotFound) {
				color.White("No exercise found for '%s'", args[0])
				return err
			}
			if err != nil {
				return err
			}
			if ex.State() == exercises.Pending && !reveal {
				color.Yellow("%s is still pending. Solve it first, or use --reveal to look anyway.", ex.Name)
				return errors.New("solution hidden")
			}
			data, err := fs.ReadFile(solutions, ex.SolutionPath())
			if err != nil {
				return fmt.Errorf("no solution for %s: %w", ex.Name, err)
			}
			fmt.Print(string(data))
			return nil
		},
	}
	c.Flags().BoolVar(&reveal, "reveal", false, "show the solution even if the exercise is pending")
	return c
}
