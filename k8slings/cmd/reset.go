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
		Long: `For an exercise with a file: overwrites the exercise file with the original from
the binary, which also brings back the 'I AM NOT DONE' line. Your changes to that
file are lost.

For a query exercise: the same, and its cluster state is set up again.

For a fix exercise (nothing to edit, you repair the cluster): breaks the cluster
again, as it was when you started, and forgets that you had fixed it.`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if ex, ok := fixExercise(src, args[0]); ok {
				if err := ex.Reseed(cmd.Context(), newEnv()); err != nil {
					color.Red("%s", err)
					return err
				}
				printSeeded(ex)
				return nil
			}
			p, err := workspace.Reset(src, root, args[0])
			if errors.Is(err, exercises.ErrExerciseNotFound) {
				color.White("No exercise found for '%s'", args[0])
				return err
			}
			if err != nil {
				return err
			}
			color.Green("Restored %s", p)
			if ex, ok := queryExercise(src, args[0]); ok && !ex.Offline {
				if err := ex.Reseed(cmd.Context(), newEnv()); err != nil {
					color.Yellow("%s", err)
					return nil // the file is back; the state is made on the next run
				}
				color.White("The cluster state of %s is set up again.", ex.Name)
			}
			return nil
		},
	}
}

// queryExercise finds the named exercise in the embedded manifest, if it is a query exercise.
func queryExercise(src fs.FS, name string) (exercises.Exercise, bool) {
	data, err := fs.ReadFile(src, "info.toml")
	if err != nil {
		return exercises.Exercise{}, false
	}
	all, err := exercises.ParseInfo(data)
	if err != nil {
		return exercises.Exercise{}, false
	}
	for _, ex := range all {
		if ex.Name == name && ex.Mode == exercises.ModeQuery {
			return ex, true
		}
	}
	return exercises.Exercise{}, false
}

// fixExercise finds the named exercise in the embedded manifest, if it is a fix exercise.
func fixExercise(src fs.FS, name string) (exercises.Exercise, bool) {
	data, err := fs.ReadFile(src, "info.toml")
	if err != nil {
		return exercises.Exercise{}, false
	}
	all, err := exercises.ParseInfo(data)
	if err != nil {
		return exercises.Exercise{}, false
	}
	for _, ex := range all {
		if ex.Name == name && ex.Mode == exercises.ModeFix {
			return ex, true
		}
	}
	return exercises.Exercise{}, false
}
