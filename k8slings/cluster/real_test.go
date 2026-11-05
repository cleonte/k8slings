//go:build cluster

package cluster

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRealCluster creates a real kind cluster (named k8slings-test, so it never
// collides with a learner's), and checks the safety rules against it. It needs
// Docker, kind and kubectl, takes a few minutes and is run by `make
// cluster-test` and by the cluster job in CI:
//
//	go test -tags cluster -run TestRealCluster -timeout 20m ./k8slings/cluster/
func TestRealCluster(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	home, _ := os.UserHomeDir()
	userConfig := filepath.Join(home, ".kube", "config")
	before := fingerprint(userConfig)

	c := &Cluster{Name: "k8slings-test", Kubeconfig: filepath.Join(t.TempDir(), "kubeconfig"), Runner: ExecRunner{}}
	t.Cleanup(func() {
		if err := c.Down(context.Background()); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})
	_ = c.Down(ctx) // a leftover of an earlier, aborted run

	var out bytes.Buffer
	if err := c.Up(ctx, UpOptions{Out: &out}); err != nil {
		t.Fatalf("Up: %v\n%s", err, out.String())
	}
	t.Log(out.String())

	// Our own kubeconfig leads to our own API server.
	if err := c.Verify(ctx); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	st, err := c.Status(ctx)
	if err != nil || !st.Exists || st.Verified != nil || len(st.Nodes) != Nodes(false) {
		t.Fatalf("Status = %+v, %v", st, err)
	}
	for _, n := range st.Nodes {
		if !n.Ready {
			t.Errorf("node %s is not Ready", n.Name)
		}
	}
	if got, err := c.Kubectl(ctx, "get", "nodes", "-o", "name"); err != nil || strings.Count(got, "node/") != 3 {
		t.Fatalf("kubectl get nodes = %q, %v", got, err)
	}

	// A kubeconfig that leads somewhere else is refused, even with the right context name.
	other := *c
	other.Kubeconfig = filepath.Join(t.TempDir(), "kubeconfig")
	data, _ := os.ReadFile(c.Kubeconfig)
	server, _ := serverOf(data, c.Context())
	tampered := strings.Replace(string(data), server, "https://127.0.0.1:1", 1)
	if err := os.WriteFile(other.Kubeconfig, []byte(tampered), 0o600); err != nil {
		t.Fatal(err)
	}
	var wrong WrongClusterError
	if err := other.Verify(ctx); !errors.As(err, &wrong) {
		t.Fatalf("a kubeconfig with another port must be refused, got %v", err)
	}

	// Running Up again reuses the cluster.
	out.Reset()
	if err := c.Up(ctx, UpOptions{Out: &out}); err != nil || !strings.Contains(out.String(), "reusing") {
		t.Fatalf("second Up: %v\n%s", err, out.String())
	}

	if err := c.Down(ctx); err != nil {
		t.Fatalf("Down: %v", err)
	}
	if got, _ := output(ctx, ExecRunner{}, nil, "docker", "ps", "-a", "--filter", "name="+c.ControlPlane(), "--format", "{{.Names}}"); got != "" {
		t.Errorf("the control-plane container is still there: %s", got)
	}
	if _, err := os.Stat(c.Kubeconfig); !os.IsNotExist(err) {
		t.Errorf("the kubeconfig should be gone")
	}

	// The user's own kubeconfig was never touched.
	if after := fingerprint(userConfig); after != before {
		t.Fatalf("%s changed while the cluster was created and deleted", userConfig)
	}
}

func fingerprint(path string) [32]byte {
	data, _ := os.ReadFile(path)
	return sha256.Sum256(data)
}
