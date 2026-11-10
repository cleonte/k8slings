package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/cleonte/k8slings"
	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/cleonte/k8slings/k8slings/schema"
	"github.com/spf13/cobra"
)

const infoFile = "info.toml"

// newEnv is what running exercises needs. The cluster is there for cluster
// exercises; without a cluster they are skipped, never failed.
func newEnv() *exercises.Env {
	env := &exercises.Env{Embedded: k8slings.FS, Solutions: k8slings.Solutions, Validator: schema.New(k8slings.Schemas)}
	if cl, err := defaultCluster(); err == nil {
		env.Kube = cl
	}
	return env
}

func NewRootCmd(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "k8slings",
		Short:         "Learn Kubernetes through small broken exercises",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}

	rootCmd.AddCommand(ClusterCmd())
	rootCmd.AddCommand(DoctorCmd(infoFile, k8slings.Schemas))
	rootCmd.AddCommand(HintCmd(infoFile))
	rootCmd.AddCommand(InitCmd(k8slings.FS))
	rootCmd.AddCommand(KubectlCmd())
	rootCmd.AddCommand(ListCmd(infoFile))
	rootCmd.AddCommand(ResetCmd(k8slings.FS, "."))
	rootCmd.AddCommand(RunCmd(infoFile))
	rootCmd.AddCommand(SolutionCmd(infoFile, k8slings.Solutions))
	rootCmd.AddCommand(UpdateCmd(k8slings.FS, "."))
	rootCmd.AddCommand(VerifyCmd(infoFile))
	rootCmd.AddCommand(WatchCmd(infoFile))

	return rootCmd
}

func Execute(version string) {
	// Ctrl-C stops a cluster exercise that is still waiting for the cluster.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := NewRootCmd(version).ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
