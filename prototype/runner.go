package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const notDoneMarker = "I AM NOT DONE"

var errNotDone = errors.New("remove the `I AM NOT DONE` line when you think you're finished")

// skipKinds are namespaced resources we never delete by hand: they are either
// controller-managed or noise.
var skipKinds = map[string]bool{
	"events": true, "events.events.k8s.io": true, "endpoints": true,
	"endpointslices.discovery.k8s.io": true, "leases.coordination.k8s.io": true,
}

var deletableKinds string

// namespacedKinds lists every deletable namespaced resource type the cluster
// knows about (cached), so new exercises never need a hand-maintained list.
func namespacedKinds() string {
	if deletableKinds != "" {
		return deletableKinds
	}
	out, err := kubectl("api-resources", "--verbs=delete", "--namespaced", "-o", "name")
	if err != nil {
		return "all,cm,secret,sa,role,rolebinding,networkpolicy,pvc"
	}
	var kinds []string
	for _, k := range strings.Fields(out) {
		if !skipKinds[k] {
			kinds = append(kinds, k)
		}
	}
	deletableKinds = strings.Join(kinds, ",")
	return deletableKinds
}

// cleanNamespace wipes everything the learner might have created so each run
// starts from the manifest alone (and renamed objects can't linger).
func cleanNamespace(ns string) {
	kubectl("create", "namespace", ns) // ignore AlreadyExists
	kubectl("delete", namespacedKinds(),
		"--all", "-n", ns, "--now", "--wait=true", "--timeout=60s")
	// Deleting sa --all removes "default"; bare Pods are rejected until the
	// controller recreates it, so wait for it.
	for i := 0; i < 30; i++ {
		if _, err := kubectl("get", "sa", "default", "-n", ns); err == nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// Run applies the learner's manifest and polls the exercise's check until it
// passes or times out. progress is called with status lines.
func Run(ws *Workspace, ex *Exercise, progress func(string)) error {
	path := ws.exercisePath(ex)
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if strings.Contains(string(b), notDoneMarker) {
		return errNotDone
	}
	if err := clusterReady(); err != nil {
		return err
	}
	ns := ex.Namespace()
	progress("Resetting namespace " + ns + " ...")
	cleanNamespace(ns)

	progress("kubectl apply -f " + relPath(ws, path))
	if out, err := kubectl("apply", "-n", ns, "-f", path); err != nil {
		return fmt.Errorf("kubectl apply failed:\n%s", indent(out))
	}

	timeout := ex.Timeout
	if s, err := strconv.Atoi(os.Getenv("K8SLINGS_TIMEOUT")); err == nil && s > 0 {
		timeout = time.Duration(s) * time.Second // lets e2e fail fast on "still broken" runs
	}

	env := &Env{NS: ns}
	deadline := time.Now().Add(timeout)
	var last error
	for {
		last = ex.Check(env)
		if last == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%v\n\n%s", last, diagnostics(ns))
		}
		progress("waiting: " + last.Error())
		time.Sleep(2 * time.Second)
	}
}

func diagnostics(ns string) string {
	var sb strings.Builder
	sb.WriteString(dim("Cluster state in namespace " + ns + ":\n"))
	out, _ := kubectl("get", "all,cm,sa,role,rolebinding", "-n", ns)
	sb.WriteString(indent(out))
	ev, _ := kubectl("get", "events", "-n", ns, "--sort-by=.lastTimestamp", "--no-headers")
	lines := strings.Split(strings.TrimSpace(ev), "\n")
	if len(lines) > 5 {
		lines = lines[len(lines)-5:]
	}
	if len(lines) > 0 && lines[0] != "" {
		sb.WriteString("\n" + dim("Recent events:\n") + indent(strings.Join(lines, "\n")))
	}
	sb.WriteString("\n\n" + dim(fmt.Sprintf("Investigate yourself: kubectl -n %s get,describe,logs ...  (context %s)", ns, kubeContext)))
	return sb.String()
}

func indent(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = "  " + l
	}
	return strings.Join(lines, "\n")
}

func relPath(ws *Workspace, p string) string {
	if strings.HasPrefix(p, ws.Root+"/") {
		return p[len(ws.Root)+1:]
	}
	return p
}

func readEmbedded(ex *Exercise, name string) string {
	b, err := exerciseFS.ReadFile("exercises/" + ex.ID + "/" + name)
	if err != nil {
		return "(no " + name + ")"
	}
	return string(b)
}
