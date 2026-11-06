package cluster

import (
	"context"
	"errors"
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
	if err := c.Seed(context.Background(), "k8slings-fix1", "fix1", [][]byte{[]byte("kind: Deployment\n")}); err != nil {
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

// seedSteps answers a multi-step seed: apply prints what it made, and rollout
// status is answered by rollout.
func seedSteps(rollout func(args []string) (string, error)) func(Cmd) (string, error) {
	return func(cmd Cmd) (string, error) {
		args := strings.Join(cmd.Args, " ")
		switch {
		case strings.Contains(args, "api-resources"):
			return "pods\nserviceaccounts\n", nil
		case strings.Contains(args, "get pods"):
			return "No resources found in ns namespace.", nil
		case strings.Contains(args, "get serviceaccount default"):
			return "default", nil
		case slices.Contains(cmd.Args, "apply"):
			return "service/web configured\ndeployment.apps/web configured\n", nil
		case slices.Contains(cmd.Args, "rollout"):
			return rollout(cmd.Args)
		}
		return "", nil
	}
}

func TestSeedWaitsForEveryStepButTheLast(t *testing.T) {
	c, f := newCluster(t, seedSteps(func([]string) (string, error) { return "successfully rolled out", nil }))
	var said []string
	ctx := WithProgress(context.Background(), func(s string) { said = append(said, s) })

	if err := c.Seed(ctx, "k8slings-fix1", "fix1", [][]byte{[]byte("a: 1"), []byte("b: 2"), []byte("c: 3")}); err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, call := range f.called("kubectl") {
		switch {
		case slices.Contains(call.Args, "apply"):
			order = append(order, "apply")
		case slices.Contains(call.Args, "rollout"):
			if !slices.Contains(call.Args, "deployment.apps/web") || !slices.Contains(call.Args, "--timeout=90s") {
				t.Errorf("rollout status args: %v", call.Args)
			}
			order = append(order, "rollout")
		}
	}
	// The Service is not waited for; the Deployment is, after steps 1 and 2 but not after the last.
	if got := strings.Join(order, " "); got != "apply rollout apply rollout apply" {
		t.Fatalf("order = %s", got)
	}
	want := "setting up fix1|setting up fix1: step 1 of 3|setting up fix1: step 2 of 3|setting up fix1: step 3 of 3"
	if got := strings.Join(said, "|"); got != want {
		t.Fatalf("progress = %q, want %q", got, want)
	}
}

func TestSeedWithOneStepWaitsForNothing(t *testing.T) {
	c, f := newCluster(t, seedSteps(func([]string) (string, error) { return "", nil }))
	if err := c.Seed(context.Background(), "k8slings-fix1", "fix1", [][]byte{[]byte("a: 1")}); err != nil {
		t.Fatal(err)
	}
	for _, call := range f.called("kubectl") {
		if slices.Contains(call.Args, "rollout") {
			t.Fatalf("a single-step setup must not wait: %v", call.Args)
		}
	}
}

func TestSeedStopsAtAStepThatDoesNotRollOut(t *testing.T) {
	c, f := newCluster(t, seedSteps(func([]string) (string, error) {
		return "error: timed out waiting for the condition", errors.New("exit status 1")
	}))
	err := c.Seed(context.Background(), "k8slings-fix1", "fix1", [][]byte{[]byte("a: 1"), []byte("b: 2")})
	if err == nil {
		t.Fatal("a step that does not roll out must fail the seeding")
	}
	for _, want := range []string{"fix1", "step 1 of 2", "did not finish rolling out", "deployment.apps/web", "timed out", "k8slings reset fix1"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error should mention %q: %v", want, err)
		}
	}
	applies := 0
	for _, call := range f.called("kubectl") {
		if slices.Contains(call.Args, "apply") {
			applies++
		}
	}
	if applies != 1 {
		t.Fatalf("the second step was applied after the first failed (%d applies)", applies)
	}
}

func TestSeedCancelledMidWayStopsCleanly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c, f := newCluster(t, seedSteps(func([]string) (string, error) {
		cancel() // the learner presses a key while the rollout is awaited
		return "", errors.New("signal: killed")
	}))
	err := c.Seed(ctx, "k8slings-fix1", "fix1", [][]byte{[]byte("a: 1"), []byte("b: 2")})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelling must return the context error, got %v", err)
	}
	applies := 0
	for _, call := range f.called("kubectl") {
		if slices.Contains(call.Args, "apply") {
			applies++
		}
	}
	if applies != 1 {
		t.Fatalf("nothing may be applied after the cancel (%d applies)", applies)
	}
}
