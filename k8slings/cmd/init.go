package cmd

import (
	"errors"
	"io/fs"
	"os"

	"github.com/cleonte/k8slings/k8slings/workspace"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const defaultWorkspace = "k8slings-exercises"

func InitCmd(src fs.FS) *cobra.Command {
	return &cobra.Command{
		Use:   "init [folder]",
		Short: "Create a folder with all the exercises",
		Long: `Creates a new folder (default: ` + defaultWorkspace + `) with all the exercises and
the info.toml that k8slings needs. Use it after installing k8slings.`,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dest := defaultWorkspace
			if len(args) == 1 {
				dest = args[0]
			}
			if _, err := os.Stat(infoFile); err == nil {
				color.Red("This folder already has an info.toml, so it is already a k8slings workspace.")
				return errors.New("already a workspace")
			}
			if err := workspace.Init(src, dest); err != nil {
				color.Red("Failed to create %s: %s", dest, err)
				return err
			}
			color.Green("Created %s with all the exercises.", dest)
			color.White("\nNext steps:\n  cd %s\n  k8slings watch\n", dest)
			return nil
		},
	}
}
