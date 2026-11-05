package cluster

import (
	"context"
	"strings"
	"testing"
)

func TestRuntimeName(t *testing.T) {
	for host, want := range map[string]string{
		"unix:///Users/me/.lima/docker/sock/docker.sock": "Lima",
		"unix:///Users/me/.colima/default/docker.sock":   "Colima",
		"unix:///Users/me/.docker/run/docker.sock":       "Docker Desktop",
		"unix:///Users/me/.rd/docker.sock":               "Rancher Desktop",
		"unix:///run/user/1000/podman/podman.sock":       "Podman",
		"unix:///var/run/docker.sock":                    "Docker",
		"":                                               "Docker",
		"tcp://10.0.0.5:2375":                            "Docker",
	} {
		if got := RuntimeName(host); got != want {
			t.Errorf("RuntimeName(%q) = %q, want %q", host, got, want)
		}
	}
}

func TestMinorSkew(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{{"v1.37.0", "v1.37.2", 0}, {"v1.37.0", "v1.36.4", 1}, {"v1.34.1", "v1.37.0", 3}} {
		if got := MinorSkew(tc.a, tc.b); got != tc.want {
			t.Errorf("MinorSkew(%s, %s) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestTargetKubernetesComesFromThePinnedImage(t *testing.T) {
	if got := TargetKubernetes(); got != "v1.37.0" {
		t.Fatalf("got %q", got)
	}
}

func TestCorePodsKeepsOnlyTheCoreComponents(t *testing.T) {
	c, _ := newCluster(t, func(Cmd) (string, error) {
		return "coredns-559f Running\nkube-proxy-abc Error\nkindnet-x Running\netcd-cp Running\nlocal-path-provisioner-75 Running\nweb-1 Running\n", nil
	})
	pods, err := c.CorePods(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, p := range pods {
		names = append(names, p.Name+"="+p.Phase)
	}
	got := strings.Join(names, ",")
	if got != "coredns-559f=Running,kube-proxy-abc=Error,kindnet-x=Running,local-path-provisioner-75=Running" {
		t.Fatalf("got %s", got)
	}
}

func TestVersionsAreParsedFromKubectlJSON(t *testing.T) {
	c, f := newCluster(t, func(cmd Cmd) (string, error) {
		return `{"clientVersion":{"gitVersion":"v1.37.2"},"serverVersion":{"gitVersion":"v1.37.0"}}`, nil
	})
	if v, err := c.KubectlVersion(context.Background()); err != nil || v != "v1.37.2" {
		t.Fatalf("client: %q %v", v, err)
	}
	if v, err := c.ServerVersion(context.Background()); err != nil || v != "v1.37.0" {
		t.Fatalf("server: %q %v", v, err)
	}
	// The client version needs no cluster, the server version must be pinned to ours.
	calls := f.called("kubectl")
	if strings.Contains(strings.Join(calls[0].Args, " "), "--context") {
		t.Errorf("--client needs no cluster: %v", calls[0].Args)
	}
	if !strings.Contains(strings.Join(calls[1].Args, " "), "--context kind-k8slings") {
		t.Errorf("the server version must come from our cluster: %v", calls[1].Args)
	}
}
