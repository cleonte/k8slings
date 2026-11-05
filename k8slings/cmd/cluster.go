package cmd

import (
	"fmt"
	"os"

	"github.com/cleonte/k8slings/k8slings/cluster"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// ClusterCmd manages the one kind cluster that k8slings uses.
func ClusterCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "cluster",
		Short: "Create, inspect and delete the local kind cluster",
		Long: `k8slings has one kind cluster of its own, called k8slings. It has its own
kubeconfig file, so your ~/.kube/config and your current context are never
touched, and k8slings never talks to any other cluster.

Manifest exercises need no cluster at all.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(clusterUpCmd(), clusterDownCmd(), clusterStatusCmd())
	return root
}

func clusterUpCmd() *cobra.Command {
	var single bool
	c := &cobra.Command{
		Use:   "up",
		Short: "Create the cluster (one control plane and two workers) and wait until it is healthy",
		Long: `Creates the kind cluster with a pinned Kubernetes node image: one control-plane
node and two workers, which scheduling and DaemonSet exercises need. Use --single
on a small machine; exercises that need several nodes are then skipped.

The first run pulls the node image and takes a few minutes. Running it again
reuses the cluster.`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := cluster.Default()
			if err != nil {
				return err
			}
			err = cl.Up(cmd.Context(), cluster.UpOptions{Single: single, Out: cmd.OutOrStdout()})
			if err != nil {
				color.Red("%s", err)
			}
			return err
		},
	}
	c.Flags().BoolVar(&single, "single", false, "create a single node instead of a control plane and two workers")
	return c
}

func clusterDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "down",
		Short:         "Delete the cluster and its kubeconfig",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := cluster.Default()
			if err != nil {
				return err
			}
			if err := cl.Down(cmd.Context()); err != nil {
				color.Red("%s", err)
				return err
			}
			color.Green("Deleted the cluster %q.", cl.Name)
			return nil
		},
	}
}

func clusterStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "status",
		Short:         "Show whether the cluster exists and is healthy",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := cluster.Default()
			if err != nil {
				return err
			}
			st, err := cl.Status(cmd.Context())
			if err != nil {
				color.Red("%s", err)
				return err
			}
			if !st.Exists {
				color.Yellow("There is no k8slings cluster. Create it with `k8slings cluster up`.")
				return cluster.ErrNoCluster
			}
			if st.Verified != nil {
				color.Red("%s", st.Verified)
				return st.Verified
			}
			color.Green("The cluster %q exists and its kubeconfig leads to its own API server.", cl.Name)
			ready := 0
			for _, n := range st.Nodes {
				mark := color.RedString("✗")
				if n.Ready {
					mark, ready = color.GreenString("✓"), ready+1
				}
				fmt.Printf("  %s %s\n", mark, n.Name)
			}
			if ready != len(st.Nodes) {
				return fmt.Errorf("%d of %d nodes are not Ready", len(st.Nodes)-ready, len(st.Nodes))
			}
			return nil
		},
	}
}

// KubectlCmd runs kubectl against the k8slings cluster and nothing else.
func KubectlCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "kubectl [kubectl arguments]",
		Short: "Run kubectl against the k8slings cluster (and only against it)",
		Long: `Runs kubectl with the k8slings kubeconfig and context, whatever your own
current context is. Flags that choose a cluster, user or credentials (--context,
--kubeconfig, --server, ...) are refused.

  k8slings kubectl get pods -A
  k8slings kubectl -n k8slings-fix01 describe pod web`,
		DisableFlagParsing: true,
		SilenceUsage:       true,
		SilenceErrors:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
				return cmd.Help()
			}
			if err := cluster.CheckArgs(args); err != nil {
				color.Red("%s", err)
				return err
			}
			cl, err := cluster.Default()
			if err != nil {
				return err
			}
			if err := cl.Verify(cmd.Context()); err != nil {
				color.Red("%s", err)
				return err
			}
			return cl.KubectlStream(cmd.Context(), os.Stdin, os.Stdout, os.Stderr, args...)
		},
	}
}
