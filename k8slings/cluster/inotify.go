package cluster

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
)

// MinInotifyInstances is the inotify instance limit a 3-node cluster is known
// to need. With the 128 that a stock Lima VM has, `kind create` reports
// success but the workers never join and kube-proxy crashes with "too many
// open files". 512 was measured to work.
const MinInotifyInstances = 512

// InotifyLimits reads the limits of the machine that runs the containers, by
// asking a container of the node image (the kernel, and so the limit, is the
// one of the Docker VM, not of the Mac).
func (c *Cluster) InotifyLimits(ctx context.Context) (instances, watches int, err error) {
	out, err := output(ctx, c.Runner, nil, "docker", "run", "--rm", "--entrypoint", "sh", NodeImage, "-c",
		"cat /proc/sys/fs/inotify/max_user_instances /proc/sys/fs/inotify/max_user_watches")
	if err != nil {
		return 0, 0, fmt.Errorf("cannot read the inotify limits: %s", firstLine(out))
	}
	// The two numbers are the last two words. Before them there can be Docker's
	// progress output, when the node image had to be pulled first (stdout and
	// stderr arrive together), which is exactly the case of a fresh machine.
	fields := strings.Fields(out)
	if len(fields) < 2 {
		return 0, 0, fmt.Errorf("unexpected inotify limits: %q", out)
	}
	if instances, err = strconv.Atoi(fields[len(fields)-2]); err != nil {
		return 0, 0, fmt.Errorf("unexpected inotify limits: %q", lastLines(out, 3))
	}
	if watches, err = strconv.Atoi(fields[len(fields)-1]); err != nil {
		return 0, 0, fmt.Errorf("unexpected inotify limits: %q", lastLines(out, 3))
	}
	return instances, watches, nil
}

// DockerHost is where the docker client connects: $DOCKER_HOST, or the
// endpoint of the current docker context.
func (c *Cluster) DockerHost(ctx context.Context) string {
	if h := os.Getenv("DOCKER_HOST"); h != "" {
		return h
	}
	out, err := output(ctx, c.Runner, nil, "docker", "context", "inspect", "--format", "{{.Endpoints.docker.Host}}")
	if err != nil {
		return ""
	}
	return out
}

const sysctlNow = "sysctl -w fs.inotify.max_user_instances=512 fs.inotify.max_user_watches=524288"
const sysctlPersist = "printf 'fs.inotify.max_user_instances=512\\nfs.inotify.max_user_watches=524288\\n' | sudo tee /etc/sysctl.d/99-k8slings.conf"

// InotifyAdvice tells the learner how to raise the limits, for the kind of
// Docker host in use: a Lima or Colima VM, or Docker on Linux.
func InotifyAdvice(dockerHost string) string {
	var run, persist string
	switch {
	case strings.Contains(dockerHost, "/.lima/"):
		vm := limaInstance(dockerHost)
		run = "limactl shell " + vm + " sudo " + sysctlNow
		persist = "limactl shell " + vm + " sh -c " + shellQuote(sysctlPersist)
	case strings.Contains(dockerHost, "/.colima/"):
		run = "colima ssh -- sudo " + sysctlNow
		persist = "colima ssh -- sh -c " + shellQuote(sysctlPersist)
	default:
		run = "sudo " + sysctlNow
		persist = sysctlPersist
	}
	return "Raise the limits now (lost when the VM restarts):\n  " + run +
		"\nand keep them across restarts:\n  " + persist
}

// limaInstance finds the instance name in a Lima socket path such as
// unix:///Users/me/.lima/docker/sock/docker.sock.
func limaInstance(dockerHost string) string {
	if u, err := url.Parse(dockerHost); err == nil && u.Path != "" {
		dockerHost = u.Path
	}
	_, after, _ := strings.Cut(dockerHost, "/.lima/")
	if name := strings.SplitN(after, "/", 2)[0]; name != "" && path.Base(name) == name {
		return name
	}
	return "docker"
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }
