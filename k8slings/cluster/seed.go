package cluster

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

// ExerciseLabel is put on the namespace of a fix exercise once it has been
// seeded, with the name of the exercise as its value. It is how k8slings knows
// the cluster is already in its broken state, and so must not seed it again
// (which would wipe the learner's repair).
const ExerciseLabel = "k8slings.io/exercise"

// Seed brings the namespace of a fix exercise to its broken state: it empties
// the namespace, labels it, and applies the setup manifest.
func (c *Cluster) Seed(ctx context.Context, ns, name string, setup []byte) error {
	if err := c.ResetNamespace(ctx, ns); err != nil {
		return err
	}
	if out, err := c.Kubectl(ctx, "label", "namespace", ns, ExerciseLabel+"="+name, "--overwrite"); err != nil {
		return fmt.Errorf("cannot label namespace %s: %s", ns, firstLine(out))
	}
	if out, err := c.Apply(ctx, ns, setup); err != nil {
		return fmt.Errorf("cannot seed %s: %s", name, firstLine(out))
	}
	return nil
}

// Seeded reports whether the namespace is in the state a seed left it in for
// this exercise. A namespace that is gone, or seeded for another exercise, is not.
func (c *Cluster) Seeded(ctx context.Context, ns, name string) bool {
	out, err := c.Kubectl(ctx, "get", "namespace", ns, "-o", `jsonpath={.metadata.labels.k8slings\.io/exercise}`)
	return err == nil && strings.TrimSpace(out) == name
}

// UID returns the UID of an object, such as "deployment/web".
func (c *Cluster) UID(ctx context.Context, ns, resource string) (string, error) {
	out, err := c.Kubectl(ctx, "get", resource, "-n", ns, "-o", "jsonpath={.metadata.uid}")
	if err != nil {
		return "", fmt.Errorf("%s: %s", resource, firstLine(out))
	}
	return strings.TrimSpace(out), nil
}

// scriptPrelude pins `kubectl` inside a solution script to the k8slings
// cluster, whatever the script (or the machine) says. The script just calls
// kubectl, and gets this one; NS is the namespace of the exercise.
const scriptPrelude = `set -eu
kubectl() { command kubectl --kubeconfig "$K8SLINGS_KUBECONFIG" --context "$K8SLINGS_CONTEXT" "$@"; }
`

// RunScript runs a kubectl script (a solution, or a shortcut that must be
// refused) against the k8slings cluster only. It is used by the tests; learners
// repair the cluster by hand.
func (c *Cluster) RunScript(ctx context.Context, ns string, script []byte) (string, error) {
	var buf bytes.Buffer
	env := append(c.env(),
		"NS="+ns,
		"K8SLINGS_KUBECONFIG="+c.Kubeconfig,
		"K8SLINGS_CONTEXT="+c.Context(),
	)
	err := c.Runner.Run(ctx, Cmd{
		Name:   "sh",
		Args:   []string{"-c", scriptPrelude + string(script)},
		Env:    env,
		Stdout: &buf,
		Stderr: &buf,
	})
	return strings.TrimSpace(buf.String()), err
}
