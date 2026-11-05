package cluster

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// script describes a fake machine for Up.
type script struct {
	kindVersion string
	inotify     string
	exists      bool
	createFails int32 // how many creates fail before one works
	creates     atomic.Int32
	// nodes and proxies are the outputs of the readiness queries, per call.
	nodes   func(call int) string
	proxies func(call int) string
	logs    string
	nodeCnt atomic.Int32
	proxCnt atomic.Int32
	created string // content of the kind config at the time of create
}

const readyNodes = "k8slings-control-plane True\nk8slings-worker True\nk8slings-worker2 True\n"
const readyProxies = "kube-proxy-a Running\nkube-proxy-b Running\nkube-proxy-c Running\n"

func healthy() *script {
	return &script{
		kindVersion: "kind v0.33.0 go1.27.0 darwin/arm64",
		inotify:     "512\n524288\n",
		nodes:       func(int) string { return readyNodes },
		proxies:     func(int) string { return readyProxies },
	}
}

func (s *script) handler(c *Cluster) func(Cmd) (string, error) {
	return func(cmd Cmd) (string, error) {
		args := strings.Join(cmd.Args, " ")
		switch cmd.Name {
		case "docker":
			switch {
			case strings.HasPrefix(args, "version"):
				return "29.9.2", nil
			case strings.HasPrefix(args, "run"):
				return s.inotify, nil
			case strings.HasPrefix(args, "port"):
				return "6443/tcp -> 127.0.0.1:41539", nil
			case strings.HasPrefix(args, "context inspect"):
				return "unix:///Users/me/.lima/docker/sock/docker.sock", nil
			}
		case "kind":
			switch {
			case args == "version":
				return s.kindVersion, nil
			case strings.HasPrefix(args, "get clusters"):
				if s.exists {
					return "k8slings\nother", nil
				}
				return "No kind clusters found.", nil
			case strings.HasPrefix(args, "create cluster"):
				cfg := cmd.Args[slices.Index(cmd.Args, "--config")+1]
				data, _ := os.ReadFile(cfg)
				s.created = string(data)
				if s.creates.Add(1) <= s.createFails {
					return "ERROR: failed to create cluster: timeout", errFake
				}
				return "Set kubectl context", writeKubeconfig(c)
			case strings.HasPrefix(args, "export kubeconfig"):
				return "", writeKubeconfig(c)
			case strings.HasPrefix(args, "delete cluster"):
				return "Deleted nodes", nil
			}
		case "kubectl":
			switch {
			case slices.Contains(cmd.Args, "nodes"):
				return s.nodes(int(s.nodeCnt.Add(1))), nil
			case slices.Contains(cmd.Args, "logs"):
				return s.logs, nil
			case slices.Contains(cmd.Args, "pods"):
				return s.proxies(int(s.proxCnt.Add(1))), nil
			}
		}
		return "unexpected: " + cmd.Name + " " + args, errFake
	}
}

func writeKubeconfig(c *Cluster) error {
	if err := os.MkdirAll(filepath.Dir(c.Kubeconfig), 0o700); err != nil {
		return err
	}
	return os.WriteFile(c.Kubeconfig, []byte(kubeconfig("https://127.0.0.1:41539", "kind-k8slings")), 0o600)
}

func up(t *testing.T, s *script, o UpOptions) (*Cluster, *fakeRunner, string, error) {
	t.Helper()
	c, f := newCluster(t, nil)
	f.handle = s.handler(c)
	var out bytes.Buffer
	o.Out = &out
	if o.Poll == 0 {
		o.Poll = time.Millisecond
	}
	if o.ReadyTimeout == 0 {
		o.ReadyTimeout = time.Second
	}
	err := c.Up(context.Background(), o)
	return c, f, out.String(), err
}

func TestUpCreatesTheClusterWithOurOwnKubeconfigAndThePinnedImage(t *testing.T) {
	s := healthy()
	c, f, out, err := up(t, s, UpOptions{})
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	create := f.called("kind create cluster")
	if len(create) != 1 {
		t.Fatalf("kind create ran %d times", len(create))
	}
	args := create[0].Args
	for flag, want := range map[string]string{"--name": "k8slings", "--image": NodeImage, "--kubeconfig": c.Kubeconfig} {
		if i := slices.Index(args, flag); i < 0 || args[i+1] != want {
			t.Errorf("%s should be %s in %v", flag, want, args)
		}
	}
	if got := strings.Count(s.created, "role:"); got != 3 {
		t.Errorf("the kind config has %d nodes, want 3:\n%s", got, s.created)
	}
	if !strings.Contains(out, "untouched") {
		t.Errorf("the output should say the user's kubeconfig is untouched:\n%s", out)
	}
}

func TestUpSingleNode(t *testing.T) {
	s := healthy()
	s.nodes = func(int) string { return "k8slings-control-plane True\n" }
	s.proxies = func(int) string { return "kube-proxy-a Running\n" }
	if _, _, out, err := up(t, s, UpOptions{Single: true}); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if strings.Count(s.created, "role:") != 1 {
		t.Errorf("single topology should have one node:\n%s", s.created)
	}
}

func TestUpReusesAnExistingClusterAndNeverCreates(t *testing.T) {
	s := healthy()
	s.exists = true
	_, f, _, err := up(t, s, UpOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if n := len(f.called("kind create")); n != 0 {
		t.Fatalf("kind create ran %d times for an existing cluster", n)
	}
	if n := len(f.called("kind export kubeconfig")); n != 1 {
		t.Fatalf("the kubeconfig should be exported into our own file once, ran %d", n)
	}
}

func TestUpRefusesLowInotifyLimitsBeforeCreating(t *testing.T) {
	s := healthy()
	s.inotify = "128\n63408\n"
	_, f, _, err := up(t, s, UpOptions{})
	if err == nil {
		t.Fatal("expected a refusal")
	}
	for _, want := range []string{"128", "512", "limactl shell docker sudo sysctl -w", "/etc/sysctl.d"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the message should contain %q:\n%v", want, err)
		}
	}
	if n := len(f.called("kind create")); n != 0 {
		t.Fatalf("kind create ran %d times despite the low limits", n)
	}
}

func TestUpRetriesAFailedCreateOnce(t *testing.T) {
	s := healthy()
	s.createFails = 1
	_, f, _, err := up(t, s, UpOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if n := len(f.called("kind create cluster")); n != 2 {
		t.Fatalf("kind create ran %d times, want 2", n)
	}
	if n := len(f.called("kind delete cluster")); n != 1 {
		t.Fatalf("the failed attempt should be cleaned up once, ran %d deletes", n)
	}
}

func TestUpKeepsTheOutputOfAFailedCreate(t *testing.T) {
	s := healthy()
	s.createFails = 5
	_, f, _, err := up(t, s, UpOptions{})
	if err == nil || !strings.Contains(err.Error(), "failed to create cluster: timeout") {
		t.Fatalf("the error should carry kind's output: %v", err)
	}
	if n := len(f.called("kind create cluster")); n != 2 {
		t.Fatalf("kind create ran %d times, want 2", n)
	}
}

func TestUpRejectsAnOldKindAndMissingTools(t *testing.T) {
	s := healthy()
	s.kindVersion = "kind v0.30.0 go1.24 linux/amd64"
	if _, _, _, err := up(t, s, UpOptions{}); err == nil || !strings.Contains(err.Error(), "v0.33.0") {
		t.Fatalf("old kind: %v", err)
	}

	c, f := newCluster(t, func(Cmd) (string, error) { return "", nil })
	f.missing = map[string]bool{"kind": true, "kubectl": true}
	err := c.Up(context.Background(), UpOptions{})
	if err == nil || !strings.Contains(err.Error(), "kind, kubectl") || !strings.Contains(err.Error(), "manifest exercises need none") {
		t.Fatalf("missing tools: %v", err)
	}
}

func TestUpDoesNotTrustKindWhenWorkersNeverJoin(t *testing.T) {
	s := healthy()
	s.nodes = func(int) string { return "k8slings-control-plane True\n" } // the failure seen in the spike
	_, _, _, err := up(t, s, UpOptions{ReadyTimeout: 30 * time.Millisecond})
	if err == nil || !strings.Contains(err.Error(), "only 1 of 3 nodes") {
		t.Fatalf("got %v", err)
	}
}

func TestUpWaitsUntilTheNodesAreReady(t *testing.T) {
	s := healthy()
	s.nodes = func(call int) string {
		if call < 4 {
			return "k8slings-control-plane True\nk8slings-worker False\nk8slings-worker2 False\n"
		}
		return readyNodes
	}
	if _, _, out, err := up(t, s, UpOptions{}); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if s.nodeCnt.Load() < 4 {
		t.Fatalf("readiness was polled %d times", s.nodeCnt.Load())
	}
}

func TestUpNamesInotifyWhenKubeProxyCrashes(t *testing.T) {
	s := healthy()
	s.proxies = func(int) string { return "kube-proxy-a Error\nkube-proxy-b Running\nkube-proxy-c Running\n" }
	s.logs = `E1104 19:37:07 run.go:72] "command failed" err="failed complete: fsnotify watcher init: too many open files"`
	_, _, _, err := up(t, s, UpOptions{ReadyTimeout: 30 * time.Millisecond})
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"kube-proxy-a is Error", "inotify", "limactl shell docker sudo sysctl"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error should contain %q:\n%v", want, err)
		}
	}
}

func TestDownDeletesOnlyOurClusterAndItsKubeconfig(t *testing.T) {
	s := healthy()
	c, f := newCluster(t, nil)
	f.handle = s.handler(c)
	if err := writeKubeconfig(c); err != nil {
		t.Fatal(err)
	}
	if err := c.Down(context.Background()); err != nil {
		t.Fatal(err)
	}
	del := f.called("kind delete cluster")
	if len(del) != 1 || !slices.Equal(del[0].Args[:4], []string{"delete", "cluster", "--name", "k8slings"}) {
		t.Fatalf("delete calls: %+v", del)
	}
	if !slices.Contains(del[0].Args, c.Kubeconfig) {
		t.Fatalf("kind must be given our kubeconfig, not the user's: %v", del[0].Args)
	}
	if _, err := os.Stat(c.Kubeconfig); !os.IsNotExist(err) {
		t.Fatal("the kubeconfig should be removed")
	}
	if err := c.Down(context.Background()); err != nil {
		t.Fatalf("down is idempotent: %v", err)
	}
}

func TestEveryKindCallGetsAnEnvironmentWithoutKUBECONFIG(t *testing.T) {
	t.Setenv("KUBECONFIG", "/home/me/.kube/prod")
	s := healthy()
	_, f, _, err := up(t, s, UpOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"kind", "kubectl"} {
		for _, call := range f.called(name) {
			for _, kv := range call.Env {
				if strings.HasPrefix(strings.ToUpper(kv), "KUBECONFIG=") {
					t.Fatalf("%s %v got %s", call.Name, call.Args, kv)
				}
			}
			if name == "kubectl" && !slices.Contains(call.Args, "--context") {
				t.Fatalf("kubectl without --context: %v", call.Args)
			}
		}
	}
}
