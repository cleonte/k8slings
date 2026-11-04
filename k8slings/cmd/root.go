package cmd

import (
	"os"

	"github.com/cleonte/k8slings"
	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/cleonte/k8slings/k8slings/schema"
	"github.com/spf13/cobra"
)

const infoFile = "info.toml"

func newEnv() *exercises.Env {
	return &exercises.Env{Embedded: k8slings.FS, Validator: schema.New(k8slings.Schemas)}
}

func NewRootCmd(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "k8slings",
		Short:         "Learn Kubernetes through small broken exercises",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}

	rootCmd.AddCommand(HintCmd(infoFile))
	rootCmd.AddCommand(InitCmd(k8slings.FS))
	rootCmd.AddCommand(ListCmd(infoFile))
	rootCmd.AddCommand(ResetCmd(k8slings.FS, "."))
	rootCmd.AddCommand(RunCmd(infoFile))
	rootCmd.AddCommand(SolutionCmd(infoFile, k8slings.Solutions))
	rootCmd.AddCommand(VerifyCmd(infoFile))
	rootCmd.AddCommand(WatchCmd(infoFile))

	return rootCmd
}

func Execute(version string) {
	if err := NewRootCmd(version).Execute(); err != nil {
		os.Exit(1)
	}
}
