package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cleonte/k8slings"
	"github.com/cleonte/k8slings/k8slings/addons"
	"github.com/cleonte/k8slings/k8slings/cluster"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// registry is the add-ons this build ships. They are embedded, so a failure to
// read them is a bug of the build, which a test catches.
func registry() []addons.Addon {
	all, err := addons.Load(k8slings.Addons)
	if err != nil {
		panic("the add-on registry is broken: " + err.Error())
	}
	return all
}

// defaultCluster is cluster.Default with the registry set.
func defaultCluster() (*cluster.Cluster, error) {
	cl, err := cluster.Default()
	if err != nil {
		return nil, err
	}
	cl.Registry = registry()
	return cl, nil
}

func clusterAddonCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "addon",
		Short: "Install the add-ons that a few exercises need (metrics-server, ...)",
		Long: `Some exercises need something on the cluster besides Kubernetes: the Metrics API for
autoscaling, a Gateway API controller for Ingress and routes. An add-on is a pinned
manifest that ships inside k8slings and is applied to the k8slings cluster only; nothing
is downloaded. Exercises that need one are skipped, with the command that installs it,
until it is ready. 'k8slings cluster down' removes everything.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(clusterAddonListCmd(), clusterAddonInstallCmd())
	return root
}

func clusterAddonListCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "list",
		Short:         "List the add-ons and whether they are installed and ready",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := defaultCluster()
			if err != nil {
				return err
			}
			var states []cluster.AddonState
			if cl.Verify(cmd.Context()) == nil {
				if states, err = cl.AddonStates(cmd.Context()); err != nil {
					color.Red("%s", err)
					return err
				}
			} else {
				for _, a := range cl.Registry {
					states = append(states, cluster.AddonState{Addon: a})
				}
				color.Yellow("There is no k8slings cluster yet: create it with `k8slings cluster up`.")
			}
			printAddons(cmd.OutOrStdout(), states)
			return nil
		},
	}
}

// printAddons writes one line per add-on.
func printAddons(w io.Writer, states []cluster.AddonState) {
	for _, st := range states {
		state := "not installed"
		switch {
		case st.Installed && st.Ready:
			state = "ready, " + st.Version
		case st.Installed:
			state = "installed, not ready, " + st.Version
		}
		fmt.Fprintf(w, "  %-16s %-26s %s\n", st.Addon.Name, state, st.Addon.Description)
	}
}

func clusterAddonInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <name>...",
		Short: "Install add-ons into the cluster",
		Long: `Installs the named add-ons, each after the ones it needs. Running it again is safe:
an add-on that is installed, ready and of this version is left alone, and one of an older version
is updated.`,
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := defaultCluster()
			if err != nil {
				return err
			}
			if err := cl.Verify(cmd.Context()); err != nil {
				color.Red("%s", err)
				color.White("Create the cluster first: k8slings cluster up")
				return err
			}
			if err := installAddons(cmd.Context(), cl, args, cmd.OutOrStdout()); err != nil {
				color.Red("%s", err)
				return err
			}
			return nil
		},
	}
}

// installAddons installs the named add-ons and what they require.
func installAddons(ctx context.Context, cl *cluster.Cluster, names []string, out io.Writer) error {
	order, err := addons.Order(cl.Registry, names)
	if err != nil {
		return err
	}
	states, err := cl.AddonStates(ctx)
	if err != nil {
		return err
	}
	for _, a := range order {
		var cur cluster.AddonState
		for _, st := range states {
			if st.Addon.Name == a.Name {
				cur = st
			}
		}
		if cur.Installed && cur.Ready && cur.Version == a.Version {
			fmt.Fprintf(out, "%s %s is already installed and ready.\n", a.Name, a.Version)
			continue
		}
		files, err := a.Files(k8slings.Addons)
		if err != nil {
			return err
		}
		if err := cl.InstallAddon(ctx, a, files, out); err != nil {
			return err
		}
	}
	return nil
}

// addonNames splits the value of --addons.
func addonNames(v string) []string {
	var out []string
	for _, n := range strings.Split(v, ",") {
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, n)
		}
	}
	return out
}
