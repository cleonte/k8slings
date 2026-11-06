package cluster

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"
	"time"
)

func namespaceFake(defaultSAAfter int) func(Cmd) (string, error) {
	gets := 0
	return func(cmd Cmd) (string, error) {
		args := strings.Join(cmd.Args, " ")
		switch {
		case strings.Contains(args, "api-resources"):
			return "pods\nservices\nevents\nendpoints\nleases.coordination.k8s.io\nserviceaccounts\ndeployments.apps\n", nil
		case strings.Contains(args, "get serviceaccount default"):
			gets++
			if gets < defaultSAAfter {
				return "NotFound", errFake
			}
			return "default", nil
		}
		return "", nil
	}
}

func TestResetNamespaceEmptiesItAndWaitsForTheDefaultServiceAccount(t *testing.T) {
	c, f := newCluster(t, namespaceFake(3))
	if err := c.ResetNamespace(context.Background(), "k8slings-pods1"); err != nil {
		t.Fatal(err)
	}
	del := f.called("kubectl --kubeconfig")
	var deleteArgs []string
	for _, call := range del {
		if slices.Contains(call.Args, "delete") && slices.Contains(call.Args, "--wait=true") {
			deleteArgs = call.Args // the one that lists the kinds
		}
	}
	joined := strings.Join(deleteArgs, " ")
	if !strings.Contains(joined, "pods,services,serviceaccounts,deployments.apps") {
		t.Errorf("every deletable kind should be listed, minus events, endpoints and leases: %s", joined)
	}
	for _, skipped := range []string{"events", "endpoints", "leases"} {
		if strings.Contains(joined, skipped) {
			t.Errorf("%s must not be deleted by hand: %s", skipped, joined)
		}
	}
	if !strings.Contains(joined, "--all -n k8slings-pods1") || !strings.Contains(joined, "--now") {
		t.Errorf("delete = %s", joined)
	}
}

func TestResetNamespaceNeverTouchesAnotherNamespaceOrAClusterScope(t *testing.T) {
	c, f := newCluster(t, namespaceFake(1))
	if err := c.ResetNamespace(context.Background(), "k8slings-x"); err != nil {
		t.Fatal(err)
	}
	for _, call := range f.called("kubectl") {
		if slices.Contains(call.Args, "delete") && !slices.Contains(call.Args, "k8slings-x") {
			t.Fatalf("a delete without the exercise namespace: %v", call.Args)
		}
	}
}

func TestResetNamespaceGivesUpWhenTheServiceAccountNeverComes(t *testing.T) {
	c, _ := newCluster(t, namespaceFake(1000))
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := c.ResetNamespace(ctx, "k8slings-x"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestApplyFeedsTheManifestOnStdin(t *testing.T) {
	var stdin string
	c, f := newCluster(t, func(cmd Cmd) (string, error) {
		if cmd.Stdin != nil {
			b := new(bytes.Buffer)
			_, _ = b.ReadFrom(cmd.Stdin)
			stdin = b.String()
		}
		return "pod/a created", nil
	})
	out, err := c.Apply(context.Background(), "k8slings-x", []byte("kind: Pod\n"))
	if err != nil || out != "pod/a created" || stdin != "kind: Pod\n" {
		t.Fatalf("out=%q err=%v stdin=%q", out, err, stdin)
	}
	call := f.called("kubectl")[0]
	if !slices.Contains(call.Args, "--context") || !slices.Contains(call.Args, "apply") {
		t.Fatalf("args = %v", call.Args)
	}
}

func TestDiagnoseListsPodsAndTheLastWarnings(t *testing.T) {
	c, _ := newCluster(t, func(cmd Cmd) (string, error) {
		args := strings.Join(cmd.Args, " ")
		switch {
		case strings.Contains(args, "get pods"):
			return "web-1   0/1   ImagePullBackOff   0   30s\nweb-2   1/1   Running   0   30s", nil
		case strings.Contains(args, "get events"):
			return "Pod/web-1: Failed to pull image\nPod/web-1: Error: ErrImagePull", nil
		}
		return "", errFake
	})
	got := strings.Join(c.Diagnose(context.Background(), "ns"), "\n")
	for _, want := range []string{"web-1  ImagePullBackOff  ready 0/1", "web-2  Running  ready 1/1", "Warnings:", "Pod/web-1: Failed to pull image"} {
		if !strings.Contains(got, want) {
			t.Errorf("diagnosis lacks %q:\n%s", want, got)
		}
	}
}

func TestGroupWarningsCollapsesTheSameMessageOnManyObjects(t *testing.T) {
	got := groupWarnings([]string{
		"Pod/batch-a: 0/3 nodes are available: 3 node(s) didn't match Pod's node affinity",
		"Pod/batch-b: 0/3 nodes are available: 3 node(s) didn't match Pod's node affinity",
		"PersistentVolumeClaim/notes: storageclass \"fast\" not found",
		"Pod/batch-c: 0/3 nodes are available: 3 node(s) didn't match Pod's node affinity",
	}, 5)
	want := []string{
		"Pod/batch-a: 0/3 nodes are available: 3 node(s) didn't match Pod's node affinity (3 times)",
		"PersistentVolumeClaim/notes: storageclass \"fast\" not found",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestGroupWarningsKeepsTheNewestDistinctMessages(t *testing.T) {
	var in []string
	for _, m := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		in = append(in, "Pod/x: "+m)
	}
	got := groupWarnings(in, 3)
	if len(got) != 3 || got[0] != "Pod/x: e" || got[2] != "Pod/x: g" {
		t.Fatalf("got %v", got)
	}
}

func TestResetNamespaceAlsoClearsOldEvents(t *testing.T) {
	c, f := newCluster(t, namespaceFake(1))
	if err := c.ResetNamespace(context.Background(), "k8slings-x"); err != nil {
		t.Fatal(err)
	}
	if n := len(f.called("kubectl --kubeconfig")); n == 0 {
		t.Fatal("no calls")
	}
	found := false
	for _, call := range f.called("kubectl") {
		if slices.Contains(call.Args, "events") && slices.Contains(call.Args, "delete") && slices.Contains(call.Args, "k8slings-x") {
			found = true
		}
	}
	if !found {
		t.Fatal("events of earlier runs must be deleted, or they show up in this run's diagnosis")
	}
}

func TestPreloadPullsIntoEveryNodeAndSkipsWhatIsAlreadyThere(t *testing.T) {
	present := map[string]bool{"k8slings-worker busybox:1.37": true}
	c, f := newCluster(t, func(cmd Cmd) (string, error) {
		args := strings.Join(cmd.Args, " ")
		switch {
		case strings.HasPrefix(args, "get nodes"):
			return "k8slings-control-plane\nk8slings-worker\n", nil
		case strings.Contains(args, "crictl inspecti"):
			if present[cmd.Args[1]+" "+cmd.Args[len(cmd.Args)-1]] {
				return "", nil
			}
			return "", errFake
		}
		return "", nil
	})
	var out bytes.Buffer
	c.Preload(context.Background(), &UpOptions{Out: &out})
	for _, want := range []string{
		"docker exec k8slings-control-plane crictl pull nginx:1.27",
		"docker exec k8slings-worker crictl pull nginx:1.27",
		"docker exec k8slings-control-plane crictl pull busybox:1.37",
	} {
		if n := len(f.called(want)); n != 1 {
			t.Errorf("%q ran %d times, want 1", want, n)
		}
	}
	if n := len(f.called("docker exec k8slings-worker crictl pull busybox")); n != 0 {
		t.Errorf("busybox is already on the worker, but it was pulled %d times", n)
	}
	if n := len(f.called("kind load")) + len(f.called("docker pull")); n != 0 {
		t.Errorf("preload must not go through the Docker host, ran %d host commands", n)
	}
}

func TestPreloadFailureIsReportedNotFatalAndStopsAtTheFirstNode(t *testing.T) {
	c, f := newCluster(t, func(cmd Cmd) (string, error) {
		args := strings.Join(cmd.Args, " ")
		switch {
		case strings.HasPrefix(args, "get nodes"):
			return "n1\nn2\n", nil
		case strings.Contains(args, "crictl pull"):
			return "toomanyrequests: rate limit", errFake
		}
		return "", errFake
	})
	var out bytes.Buffer
	c.Preload(context.Background(), &UpOptions{Out: &out})
	if !strings.Contains(out.String(), "Could not pull nginx:1.27 on n1 (toomanyrequests") || !strings.Contains(out.String(), "pull it themselves") {
		t.Fatalf("output:\n%s", out.String())
	}
	if n := len(f.called("docker exec n2 crictl pull nginx")); n != 0 {
		t.Errorf("after a failure on n1 the other nodes should be skipped, n2 was tried %d times", n)
	}
}
