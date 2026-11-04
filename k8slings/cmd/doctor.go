package cmd

import (
	"errors"
	"io/fs"

	"github.com/cleonte/k8slings/k8slings/doctor"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func DoctorCmd(infoFile string, schemas fs.FS) *cobra.Command {
	return &cobra.Command{
		Use:           "doctor",
		Short:         "Check the exercises workspace and what k8slings brings with it",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checks := doctor.Run(infoFile, doctor.DefaultEnv(schemas))
			for _, c := range checks {
				switch c.Status {
				case doctor.OK:
					color.Green("✓ %s: %s", c.Name, c.Detail)
				case doctor.Warn:
					color.Yellow("! %s: %s", c.Name, c.Detail)
				case doctor.Fail:
					color.Red("✗ %s: %s", c.Name, c.Detail)
				}
			}
			if doctor.Failed(checks) {
				return errors.New("doctor found problems")
			}
			return nil
		},
	}
}
