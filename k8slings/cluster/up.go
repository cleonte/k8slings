package cluster

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// UpOptions controls `cluster up`.
type UpOptions struct {
	// Single creates one node instead of a control plane and two workers.
	Single bool
	// Out receives progress lines. Nil discards them.
	Out io.Writer
	// ReadyTimeout is how long to wait for the nodes and kube-proxy; default 3 minutes.
	ReadyTimeout time.Duration
	// Poll is how often readiness is checked; default 2 seconds.
	Poll time.Duration
}

func (o *UpOptions) logf(format string, args ...any) {
	if o.Out != nil {
		fmt.Fprintf(o.Out, format+"\n", args...)
	}
}

// Nodes is how many nodes the topology has.
func Nodes(single bool) int {
	if single {
		return 1
	}
	return 1 + Workers
}

// kindConfig is the kind configuration for the topology. The node count is
// fixed when the cluster is created, so it is decided here and never later.
func kindConfig(single bool) string {
	cfg := "kind: Cluster\napiVersion: kind.x-k8s.io/v1alpha4\nnodes:\n  - role: control-plane\n"
	if !single {
		for range Workers {
			cfg += "  - role: worker\n"
		}
	}
	return cfg
}

// Up creates the cluster if it does not exist and waits until it is healthy:
// every node Ready and kube-proxy Running on each of them. It does not trust
// the exit code of `kind create`, which reports success for a cluster whose
// workers never joined.
func (c *Cluster) Up(ctx context.Context, o UpOptions) error {
	if err := c.checkTools(ctx); err != nil {
		return err
	}
	exists, err := c.exists(ctx)
	if err != nil {
		return err
	}

	if exists {
		o.logf("The kind cluster %q already exists, reusing it.", c.Name)
		if err := c.exportKubeconfig(ctx); err != nil {
			return err
		}
	} else {
		if err := c.preflightInotify(ctx, &o); err != nil {
			return err
		}
		if err := c.create(ctx, &o); err != nil {
			return err
		}
	}

	if err := c.Verify(ctx); err != nil {
		return err
	}
	want := Nodes(o.Single)
	if exists {
		want = 0 // an existing cluster keeps the topology it was created with
	}
	o.logf("Waiting for the nodes and kube-proxy to be ready...")
	if err := c.waitReady(ctx, want, &o); err != nil {
		return err
	}
	o.logf("The cluster %q is ready. Its kubeconfig is %s (your own ~/.kube/config is untouched).", c.Name, c.Kubeconfig)
	return nil
}

func (c *Cluster) checkTools(ctx context.Context) error {
	var missing []string
	for _, tool := range []string{"docker", "kind", "kubectl"} {
		if !c.Runner.Has(tool) {
			missing = append(missing, tool)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required tools: %s (cluster exercises need Docker, kind and kubectl; manifest exercises need none of them)", strings.Join(missing, ", "))
	}
	if out, err := output(ctx, c.Runner, nil, "docker", "version", "--format", "{{.Server.Version}}"); err != nil {
		return fmt.Errorf("cannot reach Docker (is it running?): %s", firstLine(out))
	}
	out, err := output(ctx, c.Runner, nil, "kind", "version")
	if err != nil {
		return fmt.Errorf("cannot run kind: %s", firstLine(out))
	}
	have := kindVersionRe.FindString(out)
	if have == "" {
		return fmt.Errorf("cannot read the kind version from %q", out)
	}
	if compareVersions(have, MinKindVersion) < 0 {
		return fmt.Errorf("kind %s is too old: k8slings needs %s or newer, the version that published the pinned node image", have, MinKindVersion)
	}
	return nil
}

var kindVersionRe = regexp.MustCompile(`v\d+\.\d+\.\d+`)

// compareVersions compares two vMAJOR.MINOR.PATCH strings.
func compareVersions(a, b string) int {
	pa, pb := versionParts(a), versionParts(b)
	for i := range pa {
		if pa[i] != pb[i] {
			if pa[i] < pb[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func versionParts(v string) [3]int {
	var p [3]int
	for i, s := range strings.SplitN(strings.TrimPrefix(v, "v"), ".", 3) {
		p[i], _ = strconv.Atoi(s)
	}
	return p
}

func (c *Cluster) preflightInotify(ctx context.Context, o *UpOptions) error {
	instances, _, err := c.InotifyLimits(ctx)
	if err != nil {
		o.logf("Could not check the inotify limits (%v); continuing.", err)
		return nil
	}
	if instances < MinInotifyInstances {
		return fmt.Errorf("the Docker host allows only %d inotify instances; a kind cluster needs at least %d, or its nodes fail to start (\"too many open files\").\n%s",
			instances, MinInotifyInstances, InotifyAdvice(c.DockerHost(ctx)))
	}
	return nil
}

// exists reports whether kind knows a cluster with our name.
func (c *Cluster) exists(ctx context.Context) (bool, error) {
	out, err := output(ctx, c.Runner, c.env(), "kind", "get", "clusters")
	if err != nil {
		return false, fmt.Errorf("cannot list kind clusters: %s", firstLine(out))
	}
	for _, name := range strings.Fields(out) {
		if name == c.Name {
			return true, nil
		}
	}
	return false, nil
}

func (c *Cluster) exportKubeconfig(ctx context.Context) error {
	if err := os.MkdirAll(filepath.Dir(c.Kubeconfig), 0o700); err != nil {
		return err
	}
	if out, err := output(ctx, c.Runner, c.env(), "kind", "export", "kubeconfig", "--name", c.Name, "--kubeconfig", c.Kubeconfig); err != nil {
		return fmt.Errorf("cannot write the kubeconfig: %s", firstLine(out))
	}
	return nil
}

// create runs `kind create cluster` into the cluster's own kubeconfig. A cold
// first start (pulling the node image) failed once without a clear reason, so
// a failed attempt is cleaned up and tried again once, and the last lines of
// kind's output are kept in the error.
func (c *Cluster) create(ctx context.Context, o *UpOptions) error {
	if err := os.MkdirAll(filepath.Dir(c.Kubeconfig), 0o700); err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "k8slings-kind-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	cfgFile := filepath.Join(dir, "kind.yaml")
	if err := os.WriteFile(cfgFile, []byte(kindConfig(o.Single)), 0o600); err != nil {
		return err
	}

	args := []string{"create", "cluster", "--name", c.Name, "--image", NodeImage,
		"--kubeconfig", c.Kubeconfig, "--config", cfgFile, "--wait", "180s"}

	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		o.logf("Creating the kind cluster %q (attempt %d; the first run pulls the node image, 1 to 3 minutes)...", c.Name, attempt)
		out, err := output(ctx, c.Runner, c.env(), "kind", args...)
		if err == nil {
			return nil
		}
		lastErr = fmt.Errorf("kind create cluster failed:\n%s", lastLines(out, 15))
		if ctx.Err() != nil {
			return lastErr
		}
		// Remove what the failed attempt left, so the retry starts clean.
		_, _ = output(ctx, c.Runner, c.env(), "kind", "delete", "cluster", "--name", c.Name, "--kubeconfig", c.Kubeconfig)
	}
	return lastErr
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

const nodesTemplate = `{range .items[*]}{.metadata.name}{" "}{.status.conditions[?(@.type=="Ready")].status}{"\n"}{end}`
const proxiesTemplate = `{range .items[*]}{.metadata.name}{" "}{.status.phase}{"\n"}{end}`

// waitReady polls until want nodes are Ready and kube-proxy runs on each of
// them. With want == 0 it only requires that every node there is passes.
func (c *Cluster) waitReady(ctx context.Context, want int, o *UpOptions) error {
	timeout, poll := o.ReadyTimeout, o.Poll
	if timeout == 0 {
		timeout = 3 * time.Minute
	}
	if poll == 0 {
		poll = 2 * time.Second
	}
	deadline := time.Now().Add(timeout)
	var problem string
	for {
		problem = c.notReady(ctx, want)
		if problem == "" {
			return nil
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			break
		}
		select {
		case <-ctx.Done():
		case <-time.After(poll):
		}
	}
	return errors.New(problem + c.diagnose(ctx))
}

// notReady returns what is still missing, or "" when the cluster is healthy.
func (c *Cluster) notReady(ctx context.Context, want int) string {
	nodes, err := c.Kubectl(ctx, "get", "nodes", "-o", "jsonpath="+nodesTemplate)
	if err != nil {
		return "the API server does not answer: " + firstLine(nodes)
	}
	lines := strings.Fields(strings.ReplaceAll(nodes, "\n", " "))
	count := len(lines) / 2
	if want > 0 && count != want {
		return fmt.Sprintf("only %d of %d nodes have joined the cluster", count, want)
	}
	for i := 0; i+1 < len(lines); i += 2 {
		if lines[i+1] != "True" {
			return fmt.Sprintf("node %s is not Ready", lines[i])
		}
	}

	proxies, err := c.Kubectl(ctx, "get", "pods", "-n", "kube-system", "-l", "k8s-app=kube-proxy", "-o", "jsonpath="+proxiesTemplate)
	if err != nil {
		return "cannot list the kube-proxy pods: " + firstLine(proxies)
	}
	pods := strings.Fields(strings.ReplaceAll(proxies, "\n", " "))
	if len(pods)/2 != count {
		return fmt.Sprintf("kube-proxy runs on %d of %d nodes", len(pods)/2, count)
	}
	for i := 0; i+1 < len(pods); i += 2 {
		if pods[i+1] != "Running" {
			return fmt.Sprintf("%s is %s, not Running", pods[i], pods[i+1])
		}
	}
	return ""
}

// diagnose adds the likely cause to a readiness failure.
func (c *Cluster) diagnose(ctx context.Context) string {
	logs, _ := c.Kubectl(ctx, "logs", "-n", "kube-system", "-l", "k8s-app=kube-proxy", "--tail=5")
	if strings.Contains(logs, "too many open files") {
		return "\nkube-proxy says \"too many open files\": the Docker host's inotify limits are too low.\n" + InotifyAdvice(c.DockerHost(ctx))
	}
	return ""
}

// Down deletes the k8slings cluster and its kubeconfig. It only ever deletes
// the cluster with our name.
func (c *Cluster) Down(ctx context.Context) error {
	if !c.Runner.Has("kind") {
		return errors.New("kind is not installed")
	}
	if out, err := output(ctx, c.Runner, c.env(), "kind", "delete", "cluster", "--name", c.Name, "--kubeconfig", c.Kubeconfig); err != nil {
		return fmt.Errorf("kind delete cluster failed: %s", firstLine(out))
	}
	if err := os.Remove(c.Kubeconfig); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// NodeStatus is one node of the cluster.
type NodeStatus struct {
	Name  string
	Ready bool
}

// Status is what `cluster status` reports.
type Status struct {
	Exists bool
	// Verified is nil when the kubeconfig leads to this cluster's API server.
	Verified error
	Nodes    []NodeStatus
}

// Status looks at the cluster without changing anything.
func (c *Cluster) Status(ctx context.Context) (Status, error) {
	var s Status
	if !c.Runner.Has("kind") || !c.Runner.Has("docker") {
		return s, errors.New("docker and kind are needed to look at the cluster")
	}
	var err error
	if s.Exists, err = c.exists(ctx); err != nil || !s.Exists {
		return s, err
	}
	if s.Verified = c.Verify(ctx); s.Verified != nil {
		return s, nil
	}
	out, err := c.Kubectl(ctx, "get", "nodes", "-o", "jsonpath="+nodesTemplate)
	if err != nil {
		return s, fmt.Errorf("the API server does not answer: %s", firstLine(out))
	}
	f := strings.Fields(out)
	for i := 0; i+1 < len(f); i += 2 {
		s.Nodes = append(s.Nodes, NodeStatus{Name: f[i], Ready: f[i+1] == "True"})
	}
	return s, nil
}
