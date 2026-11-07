package cluster

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"
)

// ExerciseLabel is put on the namespace of a fix exercise once it has been
// seeded, with the name of the exercise as its value. It is how k8slings knows
// the cluster is already in its broken state, and so must not seed it again
// (which would wipe the learner's repair).
const ExerciseLabel = "k8slings.io/exercise"

// SetupRolloutTimeout is how long each step of a multi-step setup, except the
// last, has to finish rolling out before the setup gives up.
const SetupRolloutTimeout = 90 * time.Second

type progressKey struct{}

// WithProgress returns a context whose long operations (seeding an exercise
// that takes several steps) report what they are doing to fn, one line at a time.
func WithProgress(ctx context.Context, fn func(string)) context.Context {
	return context.WithValue(ctx, progressKey{}, fn)
}

// Progress reports a line to whoever asked for it with WithProgress.
func Progress(ctx context.Context, format string, args ...any) {
	if fn, ok := ctx.Value(progressKey{}).(func(string)); ok && fn != nil {
		fn(fmt.Sprintf(format, args...))
	}
}

// Seed brings the namespace of a fix exercise to its broken state: it empties
// the namespace, labels it, and applies the setup steps in order. Every step
// except the last has to roll out completely before the next one is applied, so
// that the cluster has a history (a working revision 1, then a broken revision 2).
// The last step is the broken state and is not waited for. A failed or cancelled
// seed leaves the label and records nothing, which the caller treats as "not seeded".
func (c *Cluster) Seed(ctx context.Context, ns, name string, steps [][]byte) error {
	if err := checkOwned(ns); err != nil {
		return err
	}
	if len(steps) == 0 {
		return fmt.Errorf("%s has no setup", name)
	}
	Progress(ctx, "setting up %s", name)
	if err := c.ResetNamespace(ctx, ns); err != nil {
		return err
	}
	if out, err := c.Kubectl(ctx, "label", "namespace", ns, ExerciseLabel+"="+name, "--overwrite"); err != nil {
		return fmt.Errorf("cannot label namespace %s: %s", ns, firstLine(out))
	}
	for i, setup := range steps {
		if len(steps) > 1 {
			Progress(ctx, "setting up %s: step %d of %d", name, i+1, len(steps))
		}
		out, err := c.KubectlInput(ctx, strings.NewReader(string(setup)), "apply", "-n", ns, "-f", "-", "-o", "name")
		if err != nil {
			return fmt.Errorf("cannot seed %s: %s", name, firstLine(out))
		}
		if i == len(steps)-1 {
			break
		}
		for _, obj := range strings.Fields(out) {
			if !rolloutKind(obj) {
				continue
			}
			rout, err := c.Kubectl(ctx, "rollout", "status", obj, "-n", ns, fmt.Sprintf("--timeout=%ds", int(SetupRolloutTimeout/time.Second)))
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return fmt.Errorf("cannot set up %s: step %d of %d did not finish rolling out (%s): %s. Run `k8slings cluster status`, then try `k8slings reset %s`", name, i+1, len(steps), obj, firstLine(rout), name)
			}
		}
	}
	return nil
}

// rolloutKind says whether `kubectl rollout status` understands an object
// named like "deployment.apps/web".
func rolloutKind(obj string) bool {
	kind, _, _ := strings.Cut(obj, "/")
	kind, _, _ = strings.Cut(kind, ".")
	return kind == "deployment" || kind == "statefulset" || kind == "daemonset"
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
