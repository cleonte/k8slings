package cluster

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func newCluster(t *testing.T, h func(Cmd) (string, error)) (*Cluster, *fakeRunner) {
	t.Helper()
	f := &fakeRunner{handle: h}
	return &Cluster{Name: "k8slings", Kubeconfig: filepath.Join(t.TempDir(), "k8slings", "kubeconfig"), Runner: f}, f
}

func TestKubectlPinsTheClusterAndDropsKUBECONFIG(t *testing.T) {
	t.Setenv("KUBECONFIG", "/home/me/.kube/other-cluster")
	c, f := newCluster(t, func(Cmd) (string, error) { return "ok", nil })

	if _, err := c.Kubectl(context.Background(), "get", "pods", "-A"); err != nil {
		t.Fatal(err)
	}
	call := f.called("kubectl")[0]
	want := []string{"--kubeconfig", c.Kubeconfig, "--context", "kind-k8slings", "get", "pods", "-A"}
	if !slices.Equal(call.Args, want) {
		t.Fatalf("args = %v, want %v", call.Args, want)
	}
	if call.Env == nil {
		t.Fatal("the environment must be set explicitly, not inherited")
	}
	for _, kv := range call.Env {
		if strings.HasPrefix(strings.ToUpper(kv), "KUBECONFIG=") {
			t.Fatalf("KUBECONFIG leaked into the environment: %s", kv)
		}
	}
}

func TestKubectlRefusesFlagsThatChooseTheCluster(t *testing.T) {
	c, f := newCluster(t, func(Cmd) (string, error) { return "", nil })
	for _, args := range [][]string{
		{"get", "pods", "--context", "prod"},
		{"get", "pods", "--context=prod"},
		{"--kubeconfig=/other", "get", "pods"},
		{"-s", "https://prod:6443", "get", "pods"},
		{"-shttps://prod:6443", "get", "pods"},
		{"get", "pods", "-sfoo"},
		{"get", "pods", "--server=https://prod"},
		{"get", "pods", "--token", "abc"},
		{"get", "pods", "--user=admin"},
		{"get", "pods", "--cluster", "prod"},
	} {
		_, err := c.Kubectl(context.Background(), args...)
		var forbidden ErrForbiddenFlag
		if !errorsAs(err, &forbidden) {
			t.Errorf("%v: want ErrForbiddenFlag, got %v", args, err)
		}
	}
	if n := len(f.called("kubectl")); n != 0 {
		t.Fatalf("kubectl ran %d times for refused arguments", n)
	}
}

func TestFlagsAfterDoubleDashBelongToTheContainerCommand(t *testing.T) {
	c, _ := newCluster(t, func(Cmd) (string, error) { return "", nil })
	if _, err := c.Kubectl(context.Background(), "exec", "web", "--", "curl", "-s", "http://x"); err != nil {
		t.Fatalf("-s after -- is curl's flag: %v", err)
	}
	for _, args := range [][]string{
		{"get", "pods", "-n", "web", "--selector", "app=web", "-o", "wide"},
		{"get", "pods", "--sort-by=.metadata.name", "--show-labels"},
	} {
		if _, err := c.Kubectl(context.Background(), args...); err != nil {
			t.Errorf("%v must be allowed: %v", args, err)
		}
	}
}

func TestContextAndContainerNames(t *testing.T) {
	c := &Cluster{Name: "k8slings"}
	if c.Context() != "kind-k8slings" || c.ControlPlane() != "k8slings-control-plane" {
		t.Fatalf("got %s, %s", c.Context(), c.ControlPlane())
	}
}

func TestDefaultKubeconfigIsNotTheUsersOwn(t *testing.T) {
	c, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	home, _ := os.UserHomeDir()
	if c.Kubeconfig == filepath.Join(home, ".kube", "config") || !strings.Contains(c.Kubeconfig, "k8slings") {
		t.Fatalf("kubeconfig = %s", c.Kubeconfig)
	}
}

func TestKindConfigTopology(t *testing.T) {
	if got := strings.Count(kindConfig(false), "role:"); got != 3 {
		t.Errorf("default topology has %d nodes, want 3", got)
	}
	if got := strings.Count(kindConfig(true), "role:"); got != 1 {
		t.Errorf("single topology has %d nodes, want 1", got)
	}
	if !strings.Contains(kindConfig(false), "role: control-plane") {
		t.Error("no control plane")
	}
}

func TestCompareVersions(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{{"v0.33.0", "v0.33.0", 0}, {"v0.32.9", "v0.33.0", -1}, {"v0.33.1", "v0.33.0", 1}, {"v1.0.0", "v0.99.9", 1}, {"v0.9.0", "v0.10.0", -1}} {
		if got := CompareVersions(tc.a, tc.b); got != tc.want {
			t.Errorf("CompareVersions(%s, %s) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestNodeImageIsPinnedByDigest(t *testing.T) {
	if !strings.Contains(NodeImage, "@sha256:") || strings.Count(NodeImage, "sha256:") != 1 {
		t.Fatalf("the node image must be pinned by digest: %s", NodeImage)
	}
}
