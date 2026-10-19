package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

const notDoneMarker = "I AM NOT DONE"

var errNotDone = errors.New("remove the `I AM NOT DONE` line when you think you're finished")

// cleanNamespace wipes everything the learner might have created so each run
// starts from the manifest alone (and renamed objects can't linger).
func cleanNamespace(ns string) {
	kubectl("create", "namespace", ns) // ignore AlreadyExists
	kubectl("delete", "all,cm,secret,sa,role,rolebinding,networkpolicy,pvc",
		"--all", "-n", ns, "--now", "--wait=true", "--timeout=60s")
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

	env := &Env{NS: ns}
	deadline := time.Now().Add(ex.Timeout)
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
