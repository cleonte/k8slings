package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// RuntimeName says what kind of Docker host a DOCKER_HOST (or docker context
// endpoint) is, for messages and for the advice that depends on it.
func RuntimeName(dockerHost string) string {
	switch {
	case strings.Contains(dockerHost, "/.lima/"):
		return "Lima"
	case strings.Contains(dockerHost, "/.colima/"):
		return "Colima"
	case strings.Contains(dockerHost, "/.rd/"), strings.Contains(dockerHost, "rancher-desktop"):
		return "Rancher Desktop"
	case strings.Contains(dockerHost, "docker-desktop"), strings.Contains(dockerHost, "/.docker/run/docker.sock"), strings.Contains(dockerHost, "/.docker/desktop/"):
		return "Docker Desktop"
	case strings.Contains(dockerHost, "podman"):
		return "Podman"
	case dockerHost == "", strings.HasSuffix(dockerHost, "/var/run/docker.sock"):
		return "Docker"
	}
	return "Docker"
}

// DockerVersion returns the version of the Docker server.
func (c *Cluster) DockerVersion(ctx context.Context) (string, error) {
	out, err := output(ctx, c.Runner, nil, "docker", "version", "--format", "{{.Server.Version}}")
	if err != nil {
		return "", fmt.Errorf("%s", firstLine(out))
	}
	return out, nil
}

// KindVersion returns the version of kind, such as v0.33.0.
func (c *Cluster) KindVersion(ctx context.Context) (string, error) {
	out, err := output(ctx, c.Runner, nil, "kind", "version")
	if err != nil {
		return "", fmt.Errorf("%s", firstLine(out))
	}
	if v := kindVersionRe.FindString(out); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("cannot read the kind version from %q", out)
}

type kubectlVersions struct {
	ClientVersion struct {
		GitVersion string `json:"gitVersion"`
	} `json:"clientVersion"`
	ServerVersion struct {
		GitVersion string `json:"gitVersion"`
	} `json:"serverVersion"`
}

// KubectlVersion returns the version of the kubectl client. It does not talk
// to any cluster.
func (c *Cluster) KubectlVersion(ctx context.Context) (string, error) {
	out, err := output(ctx, c.Runner, c.env(), "kubectl", "version", "--client", "-o", "json")
	if err != nil {
		return "", fmt.Errorf("%s", firstLine(out))
	}
	var v kubectlVersions
	if err := json.Unmarshal([]byte(out), &v); err != nil || v.ClientVersion.GitVersion == "" {
		return "", fmt.Errorf("cannot read the kubectl version")
	}
	return v.ClientVersion.GitVersion, nil
}

// ServerVersion returns the Kubernetes version of the k8slings cluster.
func (c *Cluster) ServerVersion(ctx context.Context) (string, error) {
	out, err := c.Kubectl(ctx, "version", "-o", "json")
	if err != nil {
		return "", fmt.Errorf("the API server does not answer: %s", firstLine(out))
	}
	var v kubectlVersions
	if err := json.Unmarshal([]byte(out), &v); err != nil || v.ServerVersion.GitVersion == "" {
		return "", fmt.Errorf("cannot read the server version")
	}
	return v.ServerVersion.GitVersion, nil
}

// MinorSkew is how many minor versions apart two vMAJOR.MINOR.PATCH versions
// are (always >= 0). kubectl supports one minor version of difference to the
// API server, in either direction.
func MinorSkew(a, b string) int {
	d := versionParts(a)[1] - versionParts(b)[1]
	if d < 0 {
		d = -d
	}
	return d
}

// NodeImagePresent reports whether the node image is already on the Docker
// host, so that looking at the host (for example its inotify limits) does not
// start a pull of about a gigabyte.
func (c *Cluster) NodeImagePresent(ctx context.Context) bool {
	_, err := output(ctx, c.Runner, nil, "docker", "image", "inspect", NodeImage)
	return err == nil
}

// CorePodNames are the system components every k8slings cluster must have
// running: DNS, the proxy, the CNI and the storage provisioner.
var CorePodNames = []string{"coredns", "kube-proxy", "kindnet", "local-path-provisioner"}

// CorePod is one pod of a core component.
type CorePod struct {
	Name  string
	Phase string
}

// CorePods lists the pods of the core components, with their phase. A
// component that has no pod at all is missing from the list.
func (c *Cluster) CorePods(ctx context.Context) ([]CorePod, error) {
	out, err := c.Kubectl(ctx, "get", "pods", "-A", "-o",
		`jsonpath={range .items[*]}{.metadata.name}{" "}{.status.phase}{"\n"}{end}`)
	if err != nil {
		return nil, fmt.Errorf("cannot list the system pods: %s", firstLine(out))
	}
	var pods []CorePod
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) != 2 {
			continue
		}
		for _, core := range CorePodNames {
			if strings.HasPrefix(f[0], core) {
				pods = append(pods, CorePod{Name: f[0], Phase: f[1]})
			}
		}
	}
	return pods, nil
}

// ProxyHint explains a kube-proxy that crashed because of the inotify limits,
// or returns "" when that is not the problem.
func (c *Cluster) ProxyHint(ctx context.Context) string { return c.diagnose(ctx) }

// TargetKubernetes is the Kubernetes version of the pinned node image.
func TargetKubernetes() string { return kindVersionRe.FindString(NodeImage) }
