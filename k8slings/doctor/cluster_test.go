package doctor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cleonte/k8slings/k8slings/addons"
	"github.com/cleonte/k8slings/k8slings/cluster"
	"github.com/cleonte/k8slings/k8slings/exercises"
)

// machine is a fake computer with Docker, kind, kubectl and maybe a cluster.
type machine struct {
	missing      map[string]bool
	dockerErr    bool
	dockerHost   string
	kind         string
	kubectl      string // client version
	server       string // cluster version
	imagePresent bool
	inotify      string
	clusters     string // output of kind get clusters
	nodes        string
	pods         string
	proxyLogs    string
	port         string
	addonLabel   string // what the label and version annotation of an add-on print
	addonRollout string // replicas wanted / ready / updated of its Deployment
}

func healthy() *machine {
	return &machine{
		dockerHost: "unix:///Users/me/.lima/docker/sock/docker.sock",
		kind:       "kind v0.33.0 go1.27.0 darwin/arm64",
		kubectl:    "v1.37.2",
		server:     "v1.37.0",
		inotify:    "512\n524288\n",
		clusters:   "k8slings",
		nodes:      "k8slings-control-plane True\nk8slings-worker True\nk8slings-worker2 True\n",
		pods:       "coredns-1 Running\nkube-proxy-1 Running\nkindnet-1 Running\nlocal-path-provisioner-1 Running\netcd-1 Running\n",
		port:       "6443/tcp -> 127.0.0.1:41539",
	}
}

func (m *machine) Has(name string) bool { return !m.missing[name] }

func (m *machine) Run(_ context.Context, c cluster.Cmd) error {
	args := strings.Join(c.Args, " ")
	out, fail := "", false
	switch c.Name {
	case "docker":
		switch {
		case strings.HasPrefix(args, "version"):
			out, fail = "29.9.2", m.dockerErr
		case strings.HasPrefix(args, "context inspect"):
			out = m.dockerHost
		case strings.HasPrefix(args, "image inspect"):
			fail = !m.imagePresent
		case strings.HasPrefix(args, "run"):
			out = m.inotify
		case strings.HasPrefix(args, "port"):
			out = m.port
		}
	case "kind":
		switch {
		case args == "version":
			out = m.kind
		case strings.HasPrefix(args, "get clusters"):
			out = m.clusters
		}
	case "kubectl":
		switch {
		case strings.Contains(args, "--client"):
			out = `{"clientVersion":{"gitVersion":"` + m.kubectl + `"}}`
		case strings.Contains(args, " version "):
			out = `{"clientVersion":{"gitVersion":"` + m.kubectl + `"},"serverVersion":{"gitVersion":"` + m.server + `"}}`
		case strings.Contains(args, "get nodes"):
			out = m.nodes
		case strings.Contains(args, "get pods -A"):
			out = m.pods
		case strings.Contains(args, " logs "):
			out = m.proxyLogs
		case strings.Contains(args, "metadata.labels"):
			out = m.addonLabel
		case strings.Contains(args, "status.readyReplicas"):
			out = m.addonRollout
		}
	}
	if c.Stdout != nil {
		_, _ = c.Stdout.Write([]byte(out))
	}
	if fail {
		return errors.New("exit status 1")
	}
	return nil
}

func doctorFor(t *testing.T, m *machine, withKubeconfig bool) []Check {
	t.Helper()
	cl := &cluster.Cluster{Name: "k8slings", Kubeconfig: filepath.Join(t.TempDir(), "kubeconfig"), Runner: m}
	if withKubeconfig {
		cfg := "apiVersion: v1\nkind: Config\nclusters:\n- name: kind-k8slings\n  cluster:\n    server: https://127.0.0.1:41539\ncontexts:\n- name: kind-k8slings\n  context:\n    cluster: kind-k8slings\n"
		if err := os.WriteFile(cl.Kubeconfig, []byte(cfg), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return clusterChecks(context.Background(), cl, nil)
}

func find(checks []Check, name string) Check {
	for _, c := range checks {
		if c.Name == name {
			return c
		}
	}
	return Check{Name: "(missing) " + name}
}

func want(t *testing.T, checks []Check, name string, status Status, parts ...string) {
	t.Helper()
	c := find(checks, name)
	if c.Status != status {
		t.Errorf("%s: status %d, want %d (%s)", name, c.Status, status, c.Detail)
	}
	for _, p := range parts {
		if !strings.Contains(c.Detail, p) {
			t.Errorf("%s: detail %q lacks %q", name, c.Detail, p)
		}
	}
}

func TestHealthyCluster(t *testing.T) {
	m := healthy()
	m.imagePresent = true
	checks := doctorFor(t, m, true)
	want(t, checks, "Container runtime", OK, "Lima", "29.9.2")
	want(t, checks, "kind", OK, "v0.33.0")
	want(t, checks, "kubectl", OK, "v1.37.2")
	want(t, checks, "inotify limits", OK, "512")
	want(t, checks, "Cluster nodes", OK, "3 nodes, all Ready")
	want(t, checks, "Cluster pods", OK, "coredns", "kube-proxy", "kindnet", "local-path-provisioner")
	want(t, checks, "Kubernetes version", OK, "v1.37.0")
	if Failed(checks) || !Ready(checks, ClusterMode) {
		t.Fatalf("a healthy cluster is ready: %+v", checks)
	}
}

func TestMissingToolsAreWarningsNotFailures(t *testing.T) {
	m := healthy()
	m.missing = map[string]bool{"docker": true, "kind": true, "kubectl": true}
	checks := doctorFor(t, m, false)
	for _, name := range []string{"Container runtime", "kind", "kubectl"} {
		want(t, checks, name, Warn)
	}
	if Failed(checks) {
		t.Fatal("missing tools must not fail doctor: manifest exercises do not need them")
	}
	if c := find(checks, "Cluster"); c.Name == "Cluster" {
		t.Error("nothing about the cluster can be checked without the tools")
	}
}

func TestDockerNotRunning(t *testing.T) {
	m := healthy()
	m.dockerErr = true
	want(t, doctorFor(t, m, false), "Container runtime", Warn, "cannot reach Docker")
}

func TestOldKind(t *testing.T) {
	m := healthy()
	m.kind = "kind v0.30.0 go1.24 linux/amd64"
	want(t, doctorFor(t, m, false), "kind", Warn, "too old", "v0.33.0")
}

func TestNoClusterYetIsAWarningWithTheCommand(t *testing.T) {
	m := healthy()
	m.clusters = "No kind clusters found."
	checks := doctorFor(t, m, false)
	want(t, checks, "Cluster", Warn, "k8slings cluster up", "skipped")
	want(t, checks, "inotify limits", OK, "not checked yet") // the node image is not there, so nothing is pulled
	if Failed(checks) {
		t.Fatal("no cluster yet is not a failure")
	}
}

func TestLowInotifyLimitsComeWithTheFixForThisRuntime(t *testing.T) {
	m := healthy()
	m.imagePresent = true
	m.inotify = "128\n63408\n"
	m.clusters = ""
	want(t, doctorFor(t, m, false), "inotify limits", Warn, "128", "512", "limactl shell docker sudo sysctl -w")
}

func TestInotifyIsNotCheckedWhenItWouldPullTheNodeImage(t *testing.T) {
	m := healthy()
	m.imagePresent = false
	m.inotify = "garbage that must not be read"
	want(t, doctorFor(t, m, true), "inotify limits", OK, "not checked yet")
}

func TestNodesNotReadyIsAFailure(t *testing.T) {
	m := healthy()
	m.nodes = "k8slings-control-plane True\nk8slings-worker False\nk8slings-worker2 True\n"
	checks := doctorFor(t, m, true)
	want(t, checks, "Cluster nodes", Fail, "1 of 3")
	if !Failed(checks) {
		t.Fatal("a broken cluster fails doctor")
	}
}

func TestSingleNodeClusterIsAWarning(t *testing.T) {
	m := healthy()
	m.nodes = "k8slings-control-plane True\n"
	want(t, doctorFor(t, m, true), "Cluster nodes", Warn, "1 node", "several nodes")
}

func TestKubeProxyCrashNamesInotify(t *testing.T) {
	m := healthy()
	m.pods = "coredns-1 Running\nkube-proxy-1 Error\nkindnet-1 Running\nlocal-path-provisioner-1 Running\n"
	m.proxyLogs = `err="failed complete: fsnotify watcher init: too many open files"`
	want(t, doctorFor(t, m, true), "Cluster pods", Fail, "kube-proxy-1 is Error", "inotify")
}

func TestMissingCorePod(t *testing.T) {
	m := healthy()
	m.pods = "coredns-1 Running\nkube-proxy-1 Running\nkindnet-1 Running\n"
	want(t, doctorFor(t, m, true), "Cluster pods", Fail, "local-path-provisioner is missing", "cluster down")
}

func TestAKubeconfigThatLeadsElsewhereIsAFailure(t *testing.T) {
	m := healthy()
	m.port = "6443/tcp -> 127.0.0.1:55555"
	want(t, doctorFor(t, m, true), "Cluster", Fail, "refusing to continue")
}

func TestVersionChecks(t *testing.T) {
	m := healthy()
	m.server = "v1.36.4"
	want(t, doctorFor(t, m, true), "Kubernetes version", Warn, "targets v1.37.0", "cluster down")

	m = healthy()
	m.kubectl = "v1.34.1"
	want(t, doctorFor(t, m, true), "Kubernetes version", Warn, "one minor version")
}

func TestRunWithoutAClusterSkipsTheClusterHalf(t *testing.T) {
	infoFile := workspace(t, info)
	checks := Run(context.Background(), infoFile, DefaultEnv(schemas, nil))
	for _, c := range checks {
		if c.Group != Offline && c.Name != "Exercise modes" {
			t.Errorf("%s is in group %q; without a cluster only offline checks exist", c.Name, c.Group)
		}
	}
}

func TestClusterExercisesDoNotMakeOfflineModeNotReady(t *testing.T) {
	infoFile := workspace(t, info+"[[exercises]]\nname='b'\npath='exercises/t/b'\ntopic='t'\nmode='cluster'\nhint='h'\n")
	checks := Run(context.Background(), infoFile, Env{FileExists: func(string) bool { return true }, Schemas: schemas})
	if !Ready(checks, Offline) {
		t.Fatalf("offline mode is ready: %+v", checks)
	}
	if Ready(checks, ClusterMode) {
		t.Fatalf("cluster mode is not ready while exercises are skipped: %+v", checks)
	}
}

func TestExerciseModesFollowTheStateOfTheCluster(t *testing.T) {
	infoFile := workspace(t, info+"[[exercises]]\nname='b'\npath='exercises/t/b'\ntopic='t'\nmode='cluster'\nhint='h'\n")
	files := func(string) bool { return true }

	// A healthy cluster: the cluster exercises can run, nothing to warn about.
	m := healthy()
	cl := &cluster.Cluster{Name: "k8slings", Kubeconfig: filepath.Join(t.TempDir(), "kubeconfig"), Runner: m}
	cfg := "apiVersion: v1\nkind: Config\nclusters:\n- name: kind-k8slings\n  cluster:\n    server: https://127.0.0.1:41539\ncontexts:\n- name: kind-k8slings\n  context:\n    cluster: kind-k8slings\n"
	if err := os.WriteFile(cl.Kubeconfig, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	checks := Run(context.Background(), infoFile, Env{FileExists: files, Schemas: schemas, Cluster: cl})
	want(t, checks, "Exercise modes", OK, "1 of 2", "ready")
	if !Ready(checks, ClusterMode) {
		t.Fatalf("a healthy cluster means cluster mode is ready: %+v", checks)
	}

	// No cluster yet: they are skipped.
	m.clusters = "No kind clusters found."
	checks = Run(context.Background(), infoFile, Env{FileExists: files, Schemas: schemas, Cluster: cl})
	want(t, checks, "Exercise modes", Warn, "1 of 2", "skipped")
}

func TestDoctorReportsAddOns(t *testing.T) {
	reg := []addons.Addon{{Name: "metrics-server", Version: "v0.9.1", Namespace: "kube-system", Ready: []string{"deployment/metrics-server"}}}
	for name, tc := range map[string]struct {
		label, rollout string
		status         Status
		parts          []string
	}{
		"not installed": {"", "", OK, []string{"metrics-server not installed", "addon install"}},
		"ready":         {"metrics-server|v0.9.1", "1/1/1", OK, []string{"metrics-server v0.9.1 ready"}},
		"not ready":     {"metrics-server|v0.9.1", "1/0/1", Warn, []string{"installed but not ready"}},
		"older version": {"metrics-server|v0.9.0", "1/1/1", Warn, []string{"v0.9.0 is installed, this k8slings ships v0.9.1"}},
	} {
		t.Run(name, func(t *testing.T) {
			m := healthy()
			m.addonLabel, m.addonRollout = tc.label, tc.rollout
			cl := &cluster.Cluster{Name: "k8slings", Kubeconfig: filepath.Join(t.TempDir(), "kubeconfig"), Runner: m, Registry: reg}
			cfg := "apiVersion: v1\nkind: Config\nclusters:\n- name: kind-k8slings\n  cluster:\n    server: https://127.0.0.1:41539\ncontexts:\n- name: kind-k8slings\n  context:\n    cluster: kind-k8slings\n"
			if err := os.WriteFile(cl.Kubeconfig, []byte(cfg), 0o600); err != nil {
				t.Fatal(err)
			}
			want(t, clusterChecks(context.Background(), cl, nil), "Add-ons", tc.status, tc.parts...)
		})
	}
}

// An add-on that is not installed says which exercises it would run, so that a
// count of skipped exercises has an answer; one that only another add-on needs
// says that instead.
func TestDoctorSaysWhatAnAddOnWouldRun(t *testing.T) {
	reg := []addons.Addon{
		{Name: "gateway", Version: "v1", Namespace: "addon-traefik", Ready: []string{"deployment/traefik"}, Requires: []string{"gateway-api"}},
		{Name: "gateway-api", Version: "v1", CRDs: []string{"gateways.gateway.networking.k8s.io"}},
		{Name: "metrics-server", Version: "v1", Namespace: "kube-system", Ready: []string{"deployment/metrics-server"}},
	}
	mk := func(names ...string) []exercises.Exercise {
		var out []exercises.Exercise
		for _, n := range names {
			req := "gateway"
			if strings.HasPrefix(n, "hpa") {
				req = "metrics-server"
			}
			out = append(out, exercises.Exercise{Name: n, Requires: []string{req}})
		}
		return out
	}
	m := healthy() // nothing is installed: the label query prints nothing
	cl := &cluster.Cluster{Name: "k8slings", Kubeconfig: filepath.Join(t.TempDir(), "kubeconfig"), Runner: m, Registry: reg}
	cfg := "apiVersion: v1\nkind: Config\nclusters:\n- name: kind-k8slings\n  cluster:\n    server: https://127.0.0.1:41539\ncontexts:\n- name: kind-k8slings\n  context:\n    cluster: kind-k8slings\n"
	if err := os.WriteFile(cl.Kubeconfig, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	checks := clusterChecks(context.Background(), cl, mk("ingress4", "gatewayapi6", "gatewayapi7", "hpa5", "gatewayapi8", "gatewayapi9"))
	want(t, checks, "Add-ons", OK,
		"gateway not installed (runs ingress4, gatewayapi6, gatewayapi7, gatewayapi8 and 1 more)",
		"gateway-api not installed (installed with gateway)",
		"metrics-server not installed (runs hpa5)")
}
