package cmd

import (
	"errors"
	"fmt"

	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func VerifyCmd(infoFile string) *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Run every exercise in order, stopping at the first one that is not done",
		Long: `Runs the exercises in the order of info.toml. Exercises that cannot run here
(for example cluster exercises without a cluster) are skipped and reported.`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			all, err := exercises.List(infoFile)
			if err != nil {
				return err
			}
			env := newEnv()
			skipped := 0
			for _, ex := range all {
				// Checking everything must not break the cluster for exercises not started.
				_, err := ex.Run(exercises.NoSeed(cmd.Context()), env)
				if exercises.IsSkipped(err) {
					color.Yellow("↷ %s skipped: %s", ex.Name, err)
					skipped++
					continue
				}
				if err != nil {
					printFailure(ex, err)
					return err
				}
				if ex.State() == exercises.Pending {
					color.Yellow("%s passes, but still has the 'I AM NOT DONE' line.", ex.Name)
					return errors.New("exercise is still pending")
				}
				color.Green("✓ %s", ex.Name)
			}
			fmt.Printf("\nAll %d exercises pass", len(all)-skipped)
			if skipped > 0 {
				fmt.Printf(" (%d skipped)", skipped)
			}
			fmt.Println(".")
			return nil
		},
	}
}
