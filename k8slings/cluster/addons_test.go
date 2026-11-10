package cluster

import (
	"context"
	"strings"
	"testing"

	"github.com/cleonte/k8slings/k8slings/addons"
)

var testAddon = addons.Addon{
	Name: "metrics-server", Version: "v0.9.1", Namespace: "kube-system",
	Manifests: []string{"components.yaml"}, Ready: []string{"deployment/metrics-server"},
	Images: []string{"registry.k8s.io/metrics-server/metrics-server:v0.9.1"},
}

func TestInstallAddonAppliesServerSideMarksItAndWaits(t *testing.T) {
	f := &fakeRunner{handle: func(c Cmd) (string, error) {
		if c.Name == "kind" {
			return "k8slings-control-plane\nk8slings-worker\n", nil
		}
		if c.Name == "docker" && strings.Contains(strings.Join(c.Args, " "), "inspecti") {
			return "", errFake // the image is not there yet: it is pulled
		}
		return "", nil
	}}
	var log strings.Builder
	if err := newQueryCluster(f).InstallAddon(context.Background(), testAddon, [][]byte{[]byte("kind: ServiceAccount\n")}, &log); err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, c := range f.calls {
		line := c.Name + " " + strings.Join(c.Args, " ")
		switch {
		case strings.Contains(line, "crictl pull"):
			order = append(order, "pull")
		case strings.Contains(line, "apply"):
			order = append(order, "apply")
			for _, want := range []string{"--server-side", "--force-conflicts", "--field-manager=k8slings"} {
				if !strings.Contains(line, want) {
					t.Errorf("apply lacks %s: %s", want, line)
				}
			}
		case strings.Contains(line, " label "):
			order = append(order, "label")
			if !strings.Contains(line, "deployment/metrics-server -n kube-system k8slings.io/addon=metrics-server") {
				t.Errorf("label: %s", line)
			}
		case strings.Contains(line, " annotate "):
			order = append(order, "annotate")
		case strings.Contains(line, "rollout status"):
			order = append(order, "rollout")
		}
	}
	if got := strings.Join(order, ","); got != "pull,pull,apply,label,annotate,rollout" {
		t.Fatalf("order = %s (an image on each of two nodes, then apply, mark, wait)", got)
	}
	if !strings.Contains(log.String(), "metrics-server is ready") {
		t.Errorf("log = %q", log.String())
	}
}

func TestInstallAddonRefusesTheNamespaceOfAnExercise(t *testing.T) {
	f := &fakeRunner{handle: func(Cmd) (string, error) { return "", nil }}
	a := testAddon
	a.Namespace = "k8slings-metrics"
	if err := newQueryCluster(f).InstallAddon(context.Background(), a, nil, nil); err == nil || len(f.calls) != 0 {
		t.Fatalf("err=%v calls=%d: an add-on may not live where an exercise reset can reach it", err, len(f.calls))
	}
}

func TestInstallAddonStopsWhenAnApplyFails(t *testing.T) {
	f := &fakeRunner{handle: func(c Cmd) (string, error) {
		if strings.Contains(strings.Join(c.Args, " "), "apply") {
			return "error: boom\nmore", errFake
		}
		return "", nil
	}}
	err := newQueryCluster(f).InstallAddon(context.Background(), testAddon, [][]byte{[]byte("x")}, nil)
	if err == nil || !strings.Contains(err.Error(), "cannot apply components.yaml of metrics-server: error: boom") {
		t.Fatalf("err = %v", err)
	}
	for _, c := range f.calls {
		if strings.Contains(strings.Join(c.Args, " "), "rollout") {
			t.Error("must not wait for an add-on that was not applied")
		}
	}
}

func TestAddonStates(t *testing.T) {
	second := addons.Addon{Name: "gateway", Version: "v1", Namespace: "addon-traefik", Ready: []string{"deployment/traefik"}}
	f := &fakeRunner{handle: func(c Cmd) (string, error) {
		line := strings.Join(c.Args, " ")
		switch {
		case strings.Contains(line, "metadata.labels") && strings.Contains(line, "kube-system"):
			return "metrics-server|v0.9.1", nil
		case strings.Contains(line, "metadata.labels"):
			return "", nil // not installed: --ignore-not-found prints nothing
		case strings.Contains(line, "status.readyReplicas") && strings.Contains(line, "kube-system"):
			return "1/1/1", nil
		}
		return "", nil
	}}
	c := newQueryCluster(f)
	c.Registry = []addons.Addon{testAddon, second}
	states, err := c.AddonStates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !states[0].Installed || !states[0].Ready || states[0].Version != "v0.9.1" {
		t.Errorf("metrics-server: %+v", states[0])
	}
	if states[1].Installed || states[1].Ready {
		t.Errorf("gateway: %+v", states[1])
	}
	ready := c.ReadyAddons(context.Background())
	if !ready["metrics-server"] || ready["gateway"] {
		t.Errorf("ready = %v", ready)
	}
}

func TestAnAddonThatIsRollingOutIsNotReady(t *testing.T) {
	f := &fakeRunner{handle: func(c Cmd) (string, error) {
		line := strings.Join(c.Args, " ")
		if strings.Contains(line, "metadata.labels") {
			return "metrics-server|v0.9.1", nil
		}
		return "2/1/1", nil // two wanted, one ready
	}}
	c := newQueryCluster(f)
	c.Registry = []addons.Addon{testAddon}
	if c.ReadyAddons(context.Background())["metrics-server"] {
		t.Error("one of two replicas is not ready")
	}
}

func TestReadyAddonsOfAClusterThatCannotBeAsked(t *testing.T) {
	f := &fakeRunner{handle: func(Cmd) (string, error) { return "Unable to connect", errFake }}
	c := newQueryCluster(f)
	c.Registry = []addons.Addon{testAddon}
	got := c.ReadyAddons(context.Background())
	if v, ok := got["metrics-server"]; !ok || v {
		t.Errorf("got %v: every add-on is listed, and none is ready", got)
	}
}
