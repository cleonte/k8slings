package cluster

import (
	"bytes"
	"context"
	"io"
	"strings"
)

// Kubectl runs kubectl against the k8slings cluster only, and returns what it
// printed. It does not check that the cluster is the right one; use Verify
// first (the runners and `k8slings kubectl` do).
func (c *Cluster) Kubectl(ctx context.Context, args ...string) (string, error) {
	if err := CheckArgs(args); err != nil {
		return "", err
	}
	return output(ctx, c.Runner, c.env(), "kubectl", c.kubectlArgs(args)...)
}

// KubectlStream is Kubectl for the commands a person types: the output goes
// straight to the terminal, and the input comes from it.
func (c *Cluster) KubectlStream(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, args ...string) error {
	if err := CheckArgs(args); err != nil {
		return err
	}
	return c.Runner.Run(ctx, Cmd{
		Name: "kubectl", Args: c.kubectlArgs(args), Env: c.env(),
		Stdin: stdin, Stdout: stdout, Stderr: stderr,
	})
}

// KubectlInput is Kubectl with standard input, for `apply -f -`.
func (c *Cluster) KubectlInput(ctx context.Context, stdin io.Reader, args ...string) (string, error) {
	if err := CheckArgs(args); err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err := c.Runner.Run(ctx, Cmd{
		Name: "kubectl", Args: c.kubectlArgs(args), Env: c.env(),
		Stdin: stdin, Stdout: &buf, Stderr: &buf,
	})
	return strings.TrimSpace(buf.String()), err
}
