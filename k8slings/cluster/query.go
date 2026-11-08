package cluster

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Limits of the command a learner writes in a query exercise.
const (
	// QueryTimeout is how long such a command may run.
	QueryTimeout = 20 * time.Second
	// QueryMaxOutput is how much of its output is kept. A command that prints more
	// is still read to its end, so that kubectl does not block on a full pipe.
	QueryMaxOutput = 1 << 20
)

// ErrNoKubectl is returned by Query when kubectl is not installed.
var ErrNoKubectl = errors.New("kubectl is not installed")

// QueryResult is what a command printed. Stdout and stderr stay apart: only the
// first is compared with the expected output.
type QueryResult struct {
	Stdout, Stderr string
	// Truncated says that the output went over QueryMaxOutput and was cut.
	Truncated bool
	// TimedOut says that the command did not finish within QueryTimeout.
	TimedOut bool
}

// capWriter keeps the first max bytes written to it and discards the rest, but
// never fails: a writer that fails would leave kubectl blocked on a full pipe.
type capWriter struct {
	buf       bytes.Buffer
	max       int
	truncated bool
}

func (w *capWriter) Write(p []byte) (int, error) {
	if room := w.max - w.buf.Len(); room < len(p) {
		w.truncated = true
		if room > 0 {
			w.buf.Write(p[:room])
		}
		return len(p), nil
	}
	return w.buf.Write(p)
}

// offlineKubeconfig is a kubeconfig that leads nowhere: a command that needs no
// cluster (create ... --dry-run=client) runs with it, so that it cannot reach
// one even if it tried. Its context has the name every command line carries
// (kind-<cluster name>), which is why it is a format string. The user has a token:
// without credentials kubectl asks for a username on the terminal.
const offlineKubeconfig = `apiVersion: v1
kind: Config
clusters:
  - name: offline
    cluster:
      server: https://127.0.0.1:1
contexts:
  - name: %[1]s
    context:
      cluster: offline
      user: offline
users:
  - name: offline
    user:
      token: offline
current-context: %[1]s
`

// Query runs a read-only kubectl command that the query package has checked, and
// returns what it printed. offline commands run with a kubeconfig that leads
// nowhere; the others run against the k8slings cluster like every other
// command. The arguments still go through CheckArgs, as they do for every path.
func (c *Cluster) Query(ctx context.Context, args []string, offline bool) (QueryResult, error) {
	if err := CheckArgs(args); err != nil {
		return QueryResult{}, err
	}
	if !c.Runner.Has("kubectl") {
		return QueryResult{}, ErrNoKubectl
	}
	run := c
	if offline {
		dir, err := os.MkdirTemp("", "k8slings-offline-")
		if err != nil {
			return QueryResult{}, err
		}
		defer os.RemoveAll(dir)
		file := filepath.Join(dir, "kubeconfig")
		if err := os.WriteFile(file, []byte(fmt.Sprintf(offlineKubeconfig, c.Context())), 0o600); err != nil {
			return QueryResult{}, err
		}
		cp := *c
		cp.Kubeconfig = file
		run = &cp
	}
	ctx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()
	stdout, stderr := &capWriter{max: QueryMaxOutput}, &capWriter{max: QueryMaxOutput}
	err := run.Runner.Run(ctx, Cmd{Name: "kubectl", Args: run.kubectlArgs(args), Env: run.env(), Stdout: stdout, Stderr: stderr})
	res := QueryResult{Stdout: stdout.buf.String(), Stderr: stderr.buf.String(), Truncated: stdout.truncated}
	if err != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		res.TimedOut = true
	}
	return res, err
}
