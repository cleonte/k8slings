package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/cleonte/k8slings"
	"github.com/cleonte/k8slings/k8slings/cluster"
	"github.com/cleonte/k8slings/k8slings/doctor"
	"github.com/cleonte/k8slings/k8slings/workspace"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func DoctorCmd(infoFile string, schemas fs.FS) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check the exercises workspace, Docker, kind, kubectl and the cluster",
		Long: `Checks what k8slings needs, in two halves.

Offline mode (manifest exercises) needs nothing but this program and the exercises folder.
Cluster mode needs Docker (or Colima, Lima, ...), kind, kubectl and the k8slings cluster; when
they are missing that is only a warning, because manifest exercises do not need them.

The exit code is non-zero only when something that should work is broken.`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var cl *cluster.Cluster
			if c, err := cluster.Default(); err == nil {
				cl = c
			}
			denv := doctor.DefaultEnv(schemas, cl)
			denv.Missing = func() []string { return workspace.Missing(k8slings.FS, ".") }
			checks := doctor.Run(cmd.Context(), infoFile, denv)

			group := ""
			for _, c := range checks {
				if c.Group != group {
					group = c.Group
					color.Cyan("\n%s", group)
				}
				detail := strings.ReplaceAll(c.Detail, "\n", "\n    ")
				switch c.Status {
				case doctor.OK:
					color.Green("✓ %s: %s", c.Name, detail)
				case doctor.Warn:
					color.Yellow("! %s: %s", c.Name, detail)
				case doctor.Fail:
					color.Red("✗ %s: %s", c.Name, detail)
				}
			}

			fmt.Println()
			summary(checks, doctor.Offline, "run k8slings from the folder that holds the exercises")
			if cl != nil {
				summary(checks, doctor.ClusterMode, "manifest exercises still work; see the warnings above")
			}
			if doctor.Failed(checks) {
				return errors.New("doctor found problems")
			}
			return nil
		},
	}
}

func summary(checks []doctor.Check, group, hint string) {
	if doctor.Ready(checks, group) {
		color.Green("%s: ready", group)
		return
	}
	color.Yellow("%s: not ready (%s)", group, hint)
}
