// Package cluster manages the one kind cluster that k8slings uses, and is the
// only code that may talk to it.
//
// The rule behind everything here: k8slings only ever talks to its own
// cluster, never to the user's current context. So the cluster has its own
// kubeconfig file (kind is told to write there, never to ~/.kube/config),
// every kubectl call names that file and the context explicitly with
// $KUBECONFIG removed from its environment, and before cluster exercises run
// the API server in that file is compared with the container that kind made.
package cluster

import (
	"fmt"
	"github.com/cleonte/k8slings/k8slings/addons"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	// DefaultName is the name of the cluster, and so of its containers
	// (<name>-control-plane) and of its kubectl context (kind-<name>).
	DefaultName = "k8slings"

	// NodeImage is the Kubernetes node image, pinned by digest so the cluster is
	// the same on every machine. It is the default image of kind v0.33.0
	// (Kubernetes 1.37.0). Bump it together with MinKindVersion, the embedded
	// schemas (scripts/gen-schemas.sh) and a re-run of every exercise.
	NodeImage = "kindest/node:v1.37.0@sha256:a1ed56cfb0e7b93589bdf97c8cd566405a265939e3620fc4f5de89adff580ae5"

	// MinKindVersion is the oldest kind that published NodeImage.
	MinKindVersion = "v0.33.0"

	// Workers is how many worker nodes the default topology has, next to the
	// control plane. A single-node cluster has none.
	Workers = 2
)

// Cluster is the k8slings cluster: its name, its own kubeconfig file, and how
// to run programs.
type Cluster struct {
	Name       string
	Kubeconfig string
	Runner     Runner

	// Registry is the list of add-ons this build knows (the addons package). The
	// caller sets it; without it the cluster knows of no add-on.
	Registry []addons.Addon

	kinds *kindsCache // set by Default; without it nothing is cached
}

// Default is the cluster learners use, with its kubeconfig below the user
// configuration folder (not ~/.kube).
func Default() (*Cluster, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("cannot find the user configuration folder: %w", err)
	}
	return &Cluster{
		Name:       DefaultName,
		Kubeconfig: filepath.Join(dir, "k8slings", "kubeconfig"),
		Runner:     ExecRunner{},
		kinds:      &kindsCache{},
	}, nil
}

// Context is the kubectl context kind creates for the cluster.
func (c *Cluster) Context() string { return "kind-" + c.Name }

// ControlPlane is the name of the control-plane container.
func (c *Cluster) ControlPlane() string { return c.Name + "-control-plane" }

// env is the environment for every program we run: the user's, without
// $KUBECONFIG, so that nothing can fall back to another cluster.
func (c *Cluster) env() []string {
	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(strings.ToUpper(kv), "KUBECONFIG=") {
			continue
		}
		env = append(env, kv)
	}
	return env
}

// kubectlArgs puts the flags that pin kubectl to this cluster in front of args.
func (c *Cluster) kubectlArgs(args []string) []string {
	return append([]string{"--kubeconfig", c.Kubeconfig, "--context", c.Context()}, args...)
}

// forbiddenFlags would let a caller point kubectl somewhere else. A later flag
// wins over an earlier one, so they are refused instead of trusted.
var forbiddenFlags = []string{
	"--kubeconfig", "--context", "--cluster", "--user", "--server", "-s",
	"--token", "--certificate-authority", "--client-certificate", "--client-key",
	"--insecure-skip-tls-verify", "--as-group", "--as-uid", "--tls-server-name",
}

// ErrForbiddenFlag is returned when kubectl arguments try to choose the cluster.
type ErrForbiddenFlag struct{ Flag string }

func (e ErrForbiddenFlag) Error() string {
	return fmt.Sprintf("k8slings picks the cluster itself: %s is not allowed here", e.Flag)
}

// CheckArgs refuses arguments that select a cluster, a user or credentials.
func CheckArgs(args []string) error {
	for _, a := range args {
		if a == "--" {
			break // what follows belongs to the command run in a container
		}
		name, _, _ := strings.Cut(a, "=")
		// -shttps://other:6443 is -s with its value attached.
		if strings.HasPrefix(a, "-s") {
			name = "-s"
		}
		if slices.Contains(forbiddenFlags, name) {
			return ErrForbiddenFlag{Flag: name}
		}
	}
	return nil
}
