package doctor

import (
	"context"
	"fmt"
	"strings"

	"github.com/cleonte/k8slings/k8slings/cluster"
)

func clusterCheck(name string, status Status, format string, args ...any) Check {
	return Check{Group: ClusterMode, Name: name, Status: status, Detail: fmt.Sprintf(format, args...)}
}

// clusterChecks looks at what cluster exercises need, from Docker up to the
// pods of the cluster. Each check only runs when what it depends on is there,
// and a missing tool or a cluster that does not exist yet is a warning: it only
// means that cluster exercises are skipped.
func clusterChecks(ctx context.Context, cl *cluster.Cluster) []Check {
	var checks []Check
	add := func(c ...Check) { checks = append(checks, c...) }

	// Docker, kind and kubectl.
	toolsOK := true
	host := cl.DockerHost(ctx)
	switch {
	case !cl.Runner.Has("docker"):
		add(clusterCheck("Container runtime", Warn, "docker is not installed: cluster exercises need Docker (Docker Desktop, Colima, Lima with the Docker template, or Docker on Linux)"))
		toolsOK = false
	default:
		if v, err := cl.DockerVersion(ctx); err != nil {
			add(clusterCheck("Container runtime", Warn, "cannot reach Docker (is it running?): %s", err))
			toolsOK = false
		} else {
			add(clusterCheck("Container runtime", OK, "%s, Docker %s", cluster.RuntimeName(host), v))
		}
	}

	if !cl.Runner.Has("kind") {
		add(clusterCheck("kind", Warn, "kind is not installed (https://kind.sigs.k8s.io/), needed to create the cluster"))
		toolsOK = false
	} else if v, err := cl.KindVersion(ctx); err != nil {
		add(clusterCheck("kind", Warn, "cannot run kind: %s", err))
		toolsOK = false
	} else if versionAtLeast(v, cluster.MinKindVersion) {
		add(clusterCheck("kind", OK, "%s", v))
	} else {
		add(clusterCheck("kind", Warn, "kind %s is too old: k8slings needs %s or newer, the version that published its pinned node image", v, cluster.MinKindVersion))
		toolsOK = false
	}

	kubectlVersion := ""
	if !cl.Runner.Has("kubectl") {
		add(clusterCheck("kubectl", Warn, "kubectl is not installed (https://kubernetes.io/docs/tasks/tools/)"))
		toolsOK = false
	} else if v, err := cl.KubectlVersion(ctx); err != nil {
		add(clusterCheck("kubectl", Warn, "cannot run kubectl: %s", err))
		toolsOK = false
	} else {
		kubectlVersion = v
		add(clusterCheck("kubectl", OK, "%s", v))
	}
	if !toolsOK {
		return checks
	}

	// The limits of the Docker host, but only when the node image is already
	// there: looking must not start a pull of about a gigabyte.
	if cl.NodeImagePresent(ctx) {
		if instances, _, err := cl.InotifyLimits(ctx); err != nil {
			add(clusterCheck("inotify limits", Warn, "%s", err))
		} else if instances < cluster.MinInotifyInstances {
			add(clusterCheck("inotify limits", Warn, "the Docker host allows %d inotify instances; a kind cluster needs at least %d, or its nodes fail to start (\"too many open files\")\n%s",
				instances, cluster.MinInotifyInstances, cluster.InotifyAdvice(host)))
		} else {
			add(clusterCheck("inotify limits", OK, "%d instances (at least %d are needed)", instances, cluster.MinInotifyInstances))
		}
	} else {
		add(clusterCheck("inotify limits", OK, "not checked yet: the node image is not on this machine, and `k8slings cluster up` checks them before it creates the cluster"))
	}

	// The cluster itself.
	st, err := cl.Status(ctx)
	switch {
	case err != nil:
		add(clusterCheck("Cluster", Warn, "%s", err))
		return checks
	case !st.Exists:
		add(clusterCheck("Cluster", Warn, "there is no k8slings cluster; create it with `k8slings cluster up` (about 2 minutes and 1.5 GB of memory). Until then, cluster exercises are skipped"))
		return checks
	case st.Verified != nil:
		add(clusterCheck("Cluster", Fail, "%s", firstLine(st.Verified.Error())))
		return checks
	}

	notReady := 0
	for _, n := range st.Nodes {
		if !n.Ready {
			notReady++
		}
	}
	switch {
	case notReady > 0:
		add(clusterCheck("Cluster nodes", Fail, "%d of %d nodes are not Ready; look with `k8slings kubectl get nodes`", notReady, len(st.Nodes)))
	case len(st.Nodes) < cluster.Nodes(false):
		add(clusterCheck("Cluster nodes", Warn, "%d node, all Ready; exercises that need several nodes are skipped (recreate the cluster without --single for them)", len(st.Nodes)))
	default:
		add(clusterCheck("Cluster nodes", OK, "%d nodes, all Ready", len(st.Nodes)))
	}

	add(corePodsCheck(ctx, cl))

	if server, err := cl.ServerVersion(ctx); err == nil {
		add(versionCheck(server, kubectlVersion))
	}
	return checks
}

func corePodsCheck(ctx context.Context, cl *cluster.Cluster) Check {
	pods, err := cl.CorePods(ctx)
	if err != nil {
		return clusterCheck("Cluster pods", Fail, "%s", err)
	}
	var problems []string
	for _, core := range cluster.CorePodNames {
		found := false
		for _, p := range pods {
			if strings.HasPrefix(p.Name, core) {
				found = true
				if p.Phase != "Running" {
					problems = append(problems, fmt.Sprintf("%s is %s", p.Name, p.Phase))
				}
			}
		}
		if !found {
			problems = append(problems, core+" is missing")
		}
	}
	if len(problems) == 0 {
		return clusterCheck("Cluster pods", OK, "%s are Running", strings.Join(cluster.CorePodNames, ", "))
	}
	detail := strings.Join(problems, "; ")
	if hint := cl.ProxyHint(ctx); hint != "" {
		detail += hint
	} else {
		detail += " (try `k8slings cluster down` and `k8slings cluster up`)"
	}
	return clusterCheck("Cluster pods", Fail, "%s", detail)
}

// versionCheck compares the cluster with the Kubernetes version k8slings
// targets and with kubectl, which supports one minor version of difference.
func versionCheck(server, kubectl string) Check {
	target := cluster.TargetKubernetes()
	switch {
	case cluster.MinorSkew(server, target) != 0:
		return clusterCheck("Kubernetes version", Warn, "the cluster runs %s but k8slings targets %s (it was created by another k8slings); recreate it with `k8slings cluster down` and `k8slings cluster up`", server, target)
	case kubectl != "" && cluster.MinorSkew(server, kubectl) > 1:
		return clusterCheck("Kubernetes version", Warn, "the cluster runs %s but kubectl is %s: kubectl supports one minor version of difference", server, kubectl)
	}
	return clusterCheck("Kubernetes version", OK, "cluster %s (target %s), kubectl %s", server, target, kubectl)
}

func versionAtLeast(have, min string) bool { return cluster.CompareVersions(have, min) >= 0 }

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}
