package cluster

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cleonte/k8slings/k8slings/addons"
)

// AddonLabel is put on the main object of an installed add-on, with its name as
// the value: it is how k8slings knows the add-on is installed. The version is
// kept in AddonVersionAnnotation.
const (
	AddonLabel             = "k8slings.io/addon"
	AddonVersionAnnotation = "k8slings.io/addon-version"
)

// AddonReadyTimeout is how long an installed add-on has to roll out.
const AddonReadyTimeout = 3 * time.Minute

// AddonState says what is on the cluster of one add-on.
type AddonState struct {
	Addon     addons.Addon
	Installed bool
	// Version is the version that was installed, which can differ from the one
	// this build ships.
	Version string
	Ready   bool
}

// checkAddonNamespace refuses a namespace that exercises own. It is the other
// half of checkOwned: that function says exercise paths only ever touch
// k8slings-<name>, this one says an add-on never lives in one.
func checkAddonNamespace(ns string) error {
	if strings.HasPrefix(ns, NamespacePrefix) {
		return fmt.Errorf("refusing to install an add-on into namespace %q: namespaces named %s* belong to the exercises", ns, NamespacePrefix)
	}
	return nil
}

// scoped adds -n ns in front of args, except for a cluster-scoped object (ns "").
func scoped(ns string, args ...string) []string {
	if ns == "" {
		return args
	}
	return append([]string{"-n", ns}, args...)
}

// target splits "deployment/name" or "namespace/deployment/name" into the
// namespace (default def) and "deployment/name".
func target(def, ref string) (ns, obj string) {
	if parts := strings.Split(ref, "/"); len(parts) == 3 {
		return parts[0], parts[1] + "/" + parts[2]
	}
	return def, ref
}

// InstallAddon installs one add-on, whose requirements are installed already:
// its images are pulled into every node, its manifests are applied server-side
// (the CRDs of Gateway API are too large for the last-applied annotation of a
// client-side apply), the main object is labelled as installed, and it waits
// until everything is ready. Only add-on code applies cluster-wide objects like
// this; exercises never do.
func (c *Cluster) InstallAddon(ctx context.Context, a addons.Addon, files [][]byte, out io.Writer) error {
	if a.Namespace == "" && (len(a.Ready) > 0 || a.Gateway != "") {
		return fmt.Errorf("add-on %s has no namespace", a.Name)
	}
	if err := checkAddonNamespace(a.Namespace); err != nil {
		return err
	}
	logf := func(format string, args ...any) {
		if out != nil {
			fmt.Fprintf(out, format+"\n", args...)
		}
	}
	logf("Installing %s %s...", a.Name, a.Version)
	c.preloadImages(ctx, a.Images, logf)
	for i, f := range files {
		o, err := c.KubectlInput(ctx, strings.NewReader(string(f)), "apply", "--server-side", "--force-conflicts", "--field-manager=k8slings", "-f", "-")
		if err != nil {
			return fmt.Errorf("cannot apply %s of %s: %s", a.Manifests[i], a.Name, firstLine(o))
		}
	}
	ns, obj := a.MainObject()
	if o, err := c.Kubectl(ctx, append([]string{"label", obj}, scoped(ns, AddonLabel+"="+a.Name, "--overwrite")...)...); err != nil {
		return fmt.Errorf("cannot mark %s as installed: %s", a.Name, firstLine(o))
	}
	if o, err := c.Kubectl(ctx, append([]string{"annotate", obj}, scoped(ns, AddonVersionAnnotation+"="+a.Version, "--overwrite")...)...); err != nil {
		return fmt.Errorf("cannot mark %s as installed: %s", a.Name, firstLine(o))
	}
	for _, r := range a.Ready {
		ns, obj := target(a.Namespace, r)
		o, err := c.Kubectl(ctx, "rollout", "status", obj, "-n", ns, fmt.Sprintf("--timeout=%ds", int(AddonReadyTimeout/time.Second)))
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("%s did not become ready: %s", a.Name, firstLine(o))
		}
	}
	for _, crd := range a.CRDs {
		o, err := c.Kubectl(ctx, "wait", "--for=condition=Established", "crd/"+crd, fmt.Sprintf("--timeout=%ds", int(AddonReadyTimeout/time.Second)))
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("%s did not become ready: %s", a.Name, firstLine(o))
		}
	}
	if a.Gateway != "" {
		ns, name, _ := strings.Cut(a.Gateway, "/")
		o, err := c.Kubectl(ctx, "wait", "--for=condition=Programmed", "gateway/"+name, "-n", ns, fmt.Sprintf("--timeout=%ds", int(AddonReadyTimeout/time.Second)))
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("%s did not become ready: %s", a.Name, firstLine(o))
		}
	}
	logf("%s is ready.", a.Name)
	return nil
}

// AddonStates looks at the cluster, without changing anything, and says for each
// add-on of the registry whether it is installed and ready.
func (c *Cluster) AddonStates(ctx context.Context) ([]AddonState, error) {
	var out []AddonState
	for _, a := range c.Registry {
		st := AddonState{Addon: a}
		ns, obj := a.MainObject()
		// The label and the version annotation of the main object.
		o, err := c.Kubectl(ctx, append([]string{"get", obj}, scoped(ns, "--ignore-not-found", "-o",
			`jsonpath={.metadata.labels.k8slings\.io/addon}|{.metadata.annotations.k8slings\.io/addon-version}`)...)...)
		if err != nil {
			return nil, fmt.Errorf("cannot look for the add-on %s: %s", a.Name, firstLine(o))
		}
		label, version, _ := strings.Cut(strings.TrimSpace(o), "|")
		if label == a.Name {
			st.Installed, st.Version = true, version
			st.Ready = c.addonReady(ctx, a)
		}
		out = append(out, st)
	}
	return out, nil
}

// addonReady is true when everything the add-on waits for is ready right now.
func (c *Cluster) addonReady(ctx context.Context, a addons.Addon) bool {
	for _, r := range a.Ready {
		ns, obj := target(a.Namespace, r)
		o, err := c.Kubectl(ctx, "get", obj, "-n", ns, "-o", `jsonpath={.spec.replicas}/{.status.readyReplicas}/{.status.updatedReplicas}`)
		if err != nil {
			return false
		}
		want, rest, _ := strings.Cut(strings.TrimSpace(o), "/")
		ready, updated, _ := strings.Cut(rest, "/")
		if want == "" || want == "0" || ready != want || updated != want {
			return false
		}
	}
	for _, crd := range a.CRDs {
		o, err := c.Kubectl(ctx, "get", "crd/"+crd, "-o", `jsonpath={.status.conditions[?(@.type=="Established")].status}`)
		if err != nil || strings.TrimSpace(o) != "True" {
			return false
		}
	}
	if a.Gateway != "" {
		ns, name, _ := strings.Cut(a.Gateway, "/")
		o, err := c.Kubectl(ctx, "get", "gateway/"+name, "-n", ns, "-o", `jsonpath={.status.conditions[?(@.type=="Programmed")].status}`)
		if err != nil || strings.TrimSpace(o) != "True" {
			return false
		}
	}
	return true
}

// ReadyAddons says, for every add-on of the registry, whether it is installed and
// ready. A cluster that cannot be asked has no ready add-on.
func (c *Cluster) ReadyAddons(ctx context.Context) map[string]bool {
	out := map[string]bool{}
	for _, a := range c.Registry {
		out[a.Name] = false
	}
	states, err := c.AddonStates(ctx)
	if err != nil {
		return out
	}
	for _, s := range states {
		out[s.Addon.Name] = s.Ready
	}
	return out
}
