package cluster

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"sigs.k8s.io/yaml"
)

// ErrNoCluster means there is no k8slings cluster (yet).
var ErrNoCluster = errors.New("there is no k8slings cluster: create it with `k8slings cluster up`")

// WrongClusterError means the kubeconfig does not lead to the k8slings
// cluster's own API server. There is no flag to carry on anyway.
type WrongClusterError struct{ Reason string }

func (e WrongClusterError) Error() string {
	return "refusing to continue: " + e.Reason + "\nk8slings only talks to its own kind cluster. Recreate it with `k8slings cluster down` and `k8slings cluster up`."
}

type kubeconfigFile struct {
	Clusters []struct {
		Name    string `json:"name"`
		Cluster struct {
			Server string `json:"server"`
		} `json:"cluster"`
	} `json:"clusters"`
	Contexts []struct {
		Name    string `json:"name"`
		Context struct {
			Cluster string `json:"cluster"`
		} `json:"context"`
	} `json:"contexts"`
}

// serverOf returns the API server URL that the kubeconfig context leads to.
func serverOf(data []byte, context string) (string, error) {
	var kc kubeconfigFile
	if err := yaml.Unmarshal(data, &kc); err != nil {
		return "", fmt.Errorf("cannot read the kubeconfig: %w", err)
	}
	for _, ctx := range kc.Contexts {
		if ctx.Name != context {
			continue
		}
		for _, cl := range kc.Clusters {
			if cl.Name == ctx.Context.Cluster {
				return cl.Cluster.Server, nil
			}
		}
	}
	return "", fmt.Errorf("the kubeconfig has no context %s", context)
}

// publishedPorts reads the output of `docker port <container> 6443/tcp`, for
// example "6443/tcp -> 127.0.0.1:41539", and returns the host ports.
func publishedPorts(out string) []string {
	var ports []string
	for _, line := range strings.Split(out, "\n") {
		_, addr, found := strings.Cut(line, "->")
		if !found {
			addr = line
		}
		addr = strings.TrimSpace(addr)
		if i := strings.LastIndex(addr, ":"); i >= 0 && i < len(addr)-1 {
			ports = append(ports, addr[i+1:])
		}
	}
	return ports
}

func isLoopback(host string) bool {
	return host == "127.0.0.1" || host == "localhost" || host == "::1" || host == "[::1]"
}

// Verify checks that the kubeconfig of the k8slings cluster leads to the API
// server of the k8slings control-plane container, on this machine: the context
// has the expected name, its server is a loopback address, and its port is the
// one Docker publishes for that container. Anything else is refused.
func (c *Cluster) Verify(ctx context.Context) error {
	data, err := os.ReadFile(c.Kubeconfig)
	if errors.Is(err, os.ErrNotExist) {
		return ErrNoCluster
	}
	if err != nil {
		return err
	}
	server, err := serverOf(data, c.Context())
	if err != nil {
		return WrongClusterError{Reason: err.Error()}
	}
	u, err := url.Parse(server)
	if err != nil || u.Host == "" {
		return WrongClusterError{Reason: fmt.Sprintf("the API server address %q is not valid", server)}
	}
	if !isLoopback(u.Hostname()) {
		return WrongClusterError{Reason: fmt.Sprintf("the API server %s is not on this machine", server)}
	}

	out, err := output(ctx, c.Runner, nil, "docker", "port", c.ControlPlane(), "6443/tcp")
	if err != nil {
		return fmt.Errorf("%w (docker said: %s)", ErrNoCluster, firstLine(out))
	}
	for _, p := range publishedPorts(out) {
		if p == u.Port() {
			return nil
		}
	}
	return WrongClusterError{Reason: fmt.Sprintf("the kubeconfig points at %s, but the container %s publishes its API server elsewhere (%s)", server, c.ControlPlane(), firstLine(out))}
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}
