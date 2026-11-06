package cluster

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// skipKinds are namespaced resources never deleted by hand: they are either
// managed by controllers or noise.
var skipKinds = map[string]bool{
	"events": true, "events.events.k8s.io": true, "endpoints": true,
	"endpointslices.discovery.k8s.io": true, "leases.coordination.k8s.io": true,
}

type kindsCache struct {
	mu    sync.Mutex
	kinds string
}

// namespacedKinds lists every deletable namespaced resource type the cluster
// knows about, so that new exercises never need a hand-maintained list.
func (c *Cluster) namespacedKinds(ctx context.Context) string {
	if c.kinds != nil {
		c.kinds.mu.Lock()
		defer c.kinds.mu.Unlock()
		if c.kinds.kinds != "" {
			return c.kinds.kinds
		}
	}
	out, err := c.Kubectl(ctx, "api-resources", "--verbs=delete", "--namespaced", "-o", "name")
	if err != nil {
		return "all,cm,secret,sa,role,rolebinding,networkpolicy,pvc"
	}
	var kinds []string
	for _, k := range strings.Fields(out) {
		if !skipKinds[k] {
			kinds = append(kinds, k)
		}
	}
	joined := strings.Join(kinds, ",")
	if c.kinds != nil {
		c.kinds.kinds = joined
	}
	return joined
}

// NamespacePrefix starts the name of every namespace k8slings owns. Emptying
// or seeding a namespace is refused for any other name, so a bug can never wipe
// default or kube-system.
const NamespacePrefix = "k8slings-"

func checkOwned(ns string) error {
	if !strings.HasPrefix(ns, NamespacePrefix) || len(ns) == len(NamespacePrefix) {
		return fmt.Errorf("refusing to touch namespace %q: k8slings only manages namespaces named %s<exercise>", ns, NamespacePrefix)
	}
	return nil
}

// ResetNamespace makes ns exist and be empty, so that each run starts from the
// manifest alone and objects the learner renamed cannot linger. It keeps the
// namespace and deletes what is in it, which is much faster than deleting the
// namespace and waiting for it to disappear.
func (c *Cluster) ResetNamespace(ctx context.Context, ns string) error {
	if err := checkOwned(ns); err != nil {
		return err
	}
	_, _ = c.Kubectl(ctx, "create", "namespace", ns) // AlreadyExists is fine

	// Controllers go first, and without touching their Pods (--cascade=orphan):
	// that is instant, and from then on nothing can create new Pods. Deleting the
	// Pods in the same command as their Deployment lets the ReplicaSet create new
	// ones meanwhile, and the garbage collector removes those later, in the
	// background, after this function has returned: they then count against the
	// next run's ResourceQuota ("exceeded quota ... used: pods=4" with only two
	// Pods in sight), which was seen on CI as a Deployment stuck one Pod short.
	// Foreground cascading would also work, but the garbage collector then uses
	// the default 30 s grace period, and a Pod whose main process ignores SIGTERM
	// (busybox sleep) would make every run wait that long. The Pods are deleted
	// below, with --now.
	if out, err := c.Kubectl(ctx, "delete", controllerKinds, "--all", "-n", ns, "--cascade=orphan", "--now", "--wait=true", "--timeout=60s"); err != nil {
		return fmt.Errorf("cannot empty namespace %s: %s", ns, firstLine(out))
	}
	if out, err := c.Kubectl(ctx, "delete", c.namespacedKinds(ctx), "--all", "-n", ns, "--now", "--wait=true", "--timeout=60s"); err != nil {
		return fmt.Errorf("cannot empty namespace %s: %s", ns, firstLine(out))
	}
	if err := c.waitNoPods(ctx, ns); err != nil {
		return err
	}
	// Events of earlier runs would show up in the diagnosis of this one (a Pod
	// of the previous attempt, a claim "being deleted"), so they go too. They
	// are not in the list of kinds deleted above, because controllers own them.
	_, _ = c.Kubectl(ctx, "delete", "events", "--all", "-n", ns)
	// Deleting serviceaccounts removes "default", and Pods are rejected until
	// the controller creates it again.
	for range 60 {
		if _, err := c.Kubectl(ctx, "get", "serviceaccount", "default", "-n", ns); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return fmt.Errorf("the default ServiceAccount of namespace %s did not come back", ns)
}

// controllerKinds are the resources that create Pods.
const controllerKinds = "deployments.apps,statefulsets.apps,daemonsets.apps,replicasets.apps,jobs.batch,cronjobs.batch"

// waitNoPods waits until the namespace has no Pods at all, not even terminating
// ones: whatever is left would still count against a quota.
func (c *Cluster) waitNoPods(ctx context.Context, ns string) error {
	for range 120 {
		out, err := c.Kubectl(ctx, "get", "pods", "-n", ns, "--no-headers")
		if err == nil && strings.TrimSpace(out) == "" || err == nil && strings.HasPrefix(strings.TrimSpace(out), "No resources found") {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return fmt.Errorf("namespace %s still has Pods after it was emptied", ns)
}

// Apply applies a manifest to the namespace.
func (c *Cluster) Apply(ctx context.Context, ns string, manifest []byte) (string, error) {
	return c.KubectlInput(ctx, strings.NewReader(string(manifest)), "apply", "-n", ns, "-f", "-")
}

// Nodes returns how many nodes the cluster has.
func (c *Cluster) Nodes(ctx context.Context) (int, error) {
	out, err := c.Kubectl(ctx, "get", "nodes", "-o", "name")
	if err != nil {
		return 0, fmt.Errorf("cannot list nodes: %s", firstLine(out))
	}
	return len(strings.Fields(out)), nil
}

// Diagnose describes what is going on in a namespace, for a learner whose
// exercise does not work yet: the Pods and their states, and the last warnings.
func (c *Cluster) Diagnose(ctx context.Context, ns string) []string {
	var lines []string
	if out, err := c.Kubectl(ctx, "get", "pods", "-n", ns, "--no-headers"); err == nil && out != "" {
		lines = append(lines, "Pods:")
		for _, l := range strings.Split(out, "\n") {
			f := strings.Fields(l)
			if len(f) >= 3 {
				lines = append(lines, fmt.Sprintf("  %s  %s  ready %s", f[0], f[2], f[1]))
			}
		}
	}
	events, err := c.Kubectl(ctx, "get", "events", "-n", ns, "--field-selector", "type=Warning", "--sort-by=.lastTimestamp",
		"-o", `jsonpath={range .items[*]}{.involvedObject.kind}/{.involvedObject.name}{": "}{.message}{"\n"}{end}`)
	if err == nil && strings.TrimSpace(events) != "" {
		lines = append(lines, "Warnings:")
		for _, w := range groupWarnings(strings.Split(strings.TrimSpace(events), "\n"), 5) {
			lines = append(lines, "  "+w)
		}
	}
	return lines
}

// groupWarnings turns "Kind/name: message" lines into at most max distinct
// messages, the newest last, each naming one object and saying how many others
// had the same warning: three Pods of a Deployment that cannot be scheduled
// have one problem, not three.
func groupWarnings(events []string, max int) []string {
	type group struct {
		first string // Kind/name of the first object
		msg   string
		n     int
	}
	var groups []*group
	index := map[string]*group{}
	for _, e := range events {
		obj, msg, found := strings.Cut(e, ": ")
		if !found {
			obj, msg = "", e
		}
		g, ok := index[msg]
		if !ok {
			g = &group{first: obj, msg: msg}
			index[msg] = g
			groups = append(groups, g)
		}
		g.n++
	}
	if len(groups) > max {
		groups = groups[len(groups)-max:]
	}
	var out []string
	for _, g := range groups {
		line := g.msg
		if g.first != "" {
			line = g.first + ": " + g.msg
		}
		if g.n > 1 {
			line += fmt.Sprintf(" (%d times)", g.n)
		}
		out = append(out, line)
	}
	return out
}
