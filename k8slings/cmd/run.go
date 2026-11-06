package cmd

import (
	"context"
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
			env := newEnv()
			var ex exercises.Exercise
			var err error
			if args[0] == "next" {
				ex, err = exercises.NextPending(infoFile, env.Availability(cmd.Context()))
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

			return runExercise(cmd.Context(), env, ex)
		},
	}
}

// runExercise checks one exercise and reports the result for the learner.
func runExercise(ctx context.Context, env *exercises.Env, ex exercises.Exercise) error {
	result, err := ex.Run(ctx, env)
	if exercises.IsSkipped(err) {
		color.Yellow(err.Error())
		return err
	}
	if exercises.IsSeeded(err) {
		printSeeded(ex)
		return nil
	}
	if err != nil {
		printFailure(ex, err)
		return err
	}
	if ex.Mode == exercises.ModeFix {
		color.Green("✅ %s is repaired. Well done.\n", ex.Name)
		return nil
	}
	color.Green("✅ %s passes (%d objects checked).\n", result.Exercise.Name, result.Objects)
	if ex.State() == exercises.Pending {
		color.White("Remove the 'I AM NOT DONE' line from %s to keep going.", ex.File())
		return fmt.Errorf("exercise is still pending")
	}
	return nil
}

// runNext checks the first exercise that is not done yet.
func runNext(ctx context.Context, infoFile string) error {
	env := newEnv()
	ex, err := exercises.NextPending(infoFile, env.Availability(ctx))
	if errors.Is(err, exercises.ErrNoPending) {
		color.Green("🎉 Every exercise is done.")
		return nil
	}
	if err != nil {
		color.Red("%s", err)
		return err
	}
	return runExercise(ctx, env, ex)
}
