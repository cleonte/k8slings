package cluster

import (
	"context"
	"slices"
	"strings"
	"testing"
)

func TestSeedEmptiesLabelsAndAppliesInThatOrder(t *testing.T) {
	var stdin string
	c, f := newCluster(t, func(cmd Cmd) (string, error) {
		args := strings.Join(cmd.Args, " ")
		switch {
		case strings.Contains(args, "api-resources"):
			return "pods\nserviceaccounts\n", nil
		case strings.Contains(args, "get pods"):
			return "No resources found in ns namespace.", nil
		case strings.Contains(args, "get serviceaccount default"):
			return "default", nil
		case cmd.Stdin != nil:
			b := new(strings.Builder)
			buf := make([]byte, 256)
			for {
				n, err := cmd.Stdin.Read(buf)
				b.Write(buf[:n])
				if err != nil {
					break
				}
			}
			stdin = b.String()
		}
		return "", nil
	})
	if err := c.Seed(context.Background(), "k8slings-fix1", "fix1", []byte("kind: Deployment\n")); err != nil {
		t.Fatal(err)
	}

	var order []string
	for _, call := range f.called("kubectl") {
		switch {
		case slices.Contains(call.Args, "delete") && slices.Contains(call.Args, "--wait=true") && !slices.Contains(call.Args, "--cascade=orphan"):
			order = append(order, "empty")
		case slices.Contains(call.Args, "label"):
			order = append(order, "label")
			if !slices.Contains(call.Args, ExerciseLabel+"=fix1") || !slices.Contains(call.Args, "--overwrite") {
				t.Errorf("label args: %v", call.Args)
			}
		case slices.Contains(call.Args, "apply"):
			order = append(order, "apply")
		}
	}
	if got := strings.Join(order, " > "); got != "empty > label > apply" {
		t.Fatalf("order = %s", got)
	}
	if stdin != "kind: Deployment\n" {
		t.Fatalf("the setup manifest must be applied, got %q", stdin)
	}
}

func TestSeededOnlyForThisExerciseAndOnlyWhenTheLabelMatches(t *testing.T) {
	label := "fix1"
	c, f := newCluster(t, func(cmd Cmd) (string, error) { return label, nil })
	if !c.Seeded(context.Background(), "ns", "fix1") {
		t.Fatal("the label matches")
	}
	if c.Seeded(context.Background(), "ns", "fix2") {
		t.Fatal("a namespace seeded for another exercise is not seeded for this one")
	}
	label = ""
	if c.Seeded(context.Background(), "ns", "fix1") {
		t.Fatal("no label, not seeded")
	}
	call := f.called("kubectl")[0]
	if !slices.Contains(call.Args, `jsonpath={.metadata.labels.k8slings\.io/exercise}`) {
		t.Fatalf("args: %v", call.Args)
	}

	gone, _ := newCluster(t, func(Cmd) (string, error) { return `Error from server (NotFound): namespaces "ns" not found`, errFake })
	if gone.Seeded(context.Background(), "ns", "fix1") {
		t.Fatal("a namespace that is gone is not seeded")
	}
}

func TestUIDReadsTheObjectsIdentity(t *testing.T) {
	c, f := newCluster(t, func(Cmd) (string, error) { return "1234-abcd\n", nil })
	uid, err := c.UID(context.Background(), "ns", "deployment/web")
	if err != nil || uid != "1234-abcd" {
		t.Fatalf("uid=%q err=%v", uid, err)
	}
	if call := f.called("kubectl")[0]; !slices.Contains(call.Args, "deployment/web") || !slices.Contains(call.Args, "ns") {
		t.Fatalf("args: %v", call.Args)
	}
	bad, _ := newCluster(t, func(Cmd) (string, error) { return "Error from server (NotFound)", errFake })
	if _, err := bad.UID(context.Background(), "ns", "deployment/web"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestRunScriptPinsKubectlToOurClusterAndGivesTheNamespace(t *testing.T) {
	t.Setenv("KUBECONFIG", "/home/me/.kube/prod")
	c, f := newCluster(t, func(Cmd) (string, error) { return "script output", nil })
	out, err := c.RunScript(context.Background(), "k8slings-fix1", []byte("kubectl -n \"$NS\" set env deployment/web A=b\n"))
	if err != nil || out != "script output" {
		t.Fatalf("out=%q err=%v", out, err)
	}
	call := f.called("sh")[0]
	script := call.Args[1]
	// The script's plain `kubectl` is a function that names our kubeconfig and context.
	for _, want := range []string{`kubectl() { command kubectl --kubeconfig "$K8SLINGS_KUBECONFIG" --context "$K8SLINGS_CONTEXT" "$@"; }`, "set -eu", `kubectl -n "$NS" set env`} {
		if !strings.Contains(script, want) {
			t.Errorf("the script lacks %q:\n%s", want, script)
		}
	}
	env := strings.Join(call.Env, "\n")
	for _, want := range []string{"NS=k8slings-fix1", "K8SLINGS_KUBECONFIG=" + c.Kubeconfig, "K8SLINGS_CONTEXT=kind-k8slings"} {
		if !strings.Contains(env, want) {
			t.Errorf("env lacks %q", want)
		}
	}
	for _, kv := range call.Env {
		if strings.HasPrefix(strings.ToUpper(kv), "KUBECONFIG=") {
			t.Fatalf("KUBECONFIG leaked into the script's environment: %s", kv)
		}
	}
}
