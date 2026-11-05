package cluster

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func kubeconfig(server, ctxName string) string {
	return `apiVersion: v1
kind: Config
clusters:
- name: kind-k8slings
  cluster:
    server: ` + server + `
contexts:
- name: ` + ctxName + `
  context:
    cluster: kind-k8slings
    user: kind-k8slings
current-context: ` + ctxName + `
`
}

func withKubeconfig(t *testing.T, content string, docker func() (string, error)) *Cluster {
	t.Helper()
	c, _ := newCluster(t, func(cmd Cmd) (string, error) {
		if cmd.Name == "docker" {
			return docker()
		}
		return "", nil
	})
	if content != "" {
		if err := os.MkdirAll(filepath.Dir(c.Kubeconfig), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(c.Kubeconfig, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return c
}

func TestVerifyAcceptsTheOwnCluster(t *testing.T) {
	c := withKubeconfig(t, kubeconfig("https://127.0.0.1:41539", "kind-k8slings"),
		func() (string, error) { return "6443/tcp -> 127.0.0.1:41539\n6443/tcp -> [::]:41539", nil })
	if err := c.Verify(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyRefusesEverythingElse(t *testing.T) {
	ok := func() (string, error) { return "6443/tcp -> 127.0.0.1:41539", nil }
	for name, tc := range map[string]struct {
		kubeconfig string
		docker     func() (string, error)
	}{
		"another port":         {kubeconfig("https://127.0.0.1:6443", "kind-k8slings"), ok},
		"a remote server":      {kubeconfig("https://prod.example.com:41539", "kind-k8slings"), ok},
		"a private address":    {kubeconfig("https://10.0.0.5:41539", "kind-k8slings"), ok},
		"another context name": {kubeconfig("https://127.0.0.1:41539", "kind-lab"), ok},
		"unparsable address":   {kubeconfig("not a url", "kind-k8slings"), ok},
		"garbage kubeconfig":   {"- just\n- a list", ok},
	} {
		t.Run(name, func(t *testing.T) {
			c := withKubeconfig(t, tc.kubeconfig, tc.docker)
			err := c.Verify(context.Background())
			var wrong WrongClusterError
			if !errors.As(err, &wrong) {
				t.Fatalf("want WrongClusterError, got %v", err)
			}
			if !strings.Contains(err.Error(), "k8slings cluster down") {
				t.Errorf("the refusal should say how to recover: %v", err)
			}
		})
	}
}

func TestVerifyWithoutAClusterOrAKubeconfig(t *testing.T) {
	c := withKubeconfig(t, "", func() (string, error) { return "", nil })
	if err := c.Verify(context.Background()); !errors.Is(err, ErrNoCluster) {
		t.Fatalf("no kubeconfig: got %v", err)
	}
	c = withKubeconfig(t, kubeconfig("https://127.0.0.1:41539", "kind-k8slings"),
		func() (string, error) { return "Error: No such container: k8slings-control-plane", errFake })
	if err := c.Verify(context.Background()); !errors.Is(err, ErrNoCluster) {
		t.Fatalf("no container: got %v", err)
	}
}

func TestPublishedPorts(t *testing.T) {
	got := publishedPorts("6443/tcp -> 0.0.0.0:1234\n6443/tcp -> [::]:5678\n")
	if strings.Join(got, ",") != "1234,5678" {
		t.Fatalf("got %v", got)
	}
}
