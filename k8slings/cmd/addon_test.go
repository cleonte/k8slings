package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cleonte/k8slings/k8slings/addons"
	"github.com/cleonte/k8slings/k8slings/cluster"
)

func TestTheEmbeddedRegistryLoads(t *testing.T) {
	if len(registry()) == 0 {
		t.Fatal("no add-ons")
	}
}

// An unknown add-on name is refused before a cluster is created.
func TestClusterUpRefusesAnUnknownAddOnFirst(t *testing.T) {
	c := clusterUpCmd()
	c.SetArgs([]string{"--addons", "metrics-server,nope"})
	var out bytes.Buffer
	c.SetOut(&out)
	err := c.Execute()
	if err == nil || !strings.Contains(err.Error(), `no add-on "nope"`) || !strings.Contains(err.Error(), "metrics-server") {
		t.Fatalf("err = %v", err)
	}
}

func TestAddonNames(t *testing.T) {
	got := addonNames(" metrics-server, gateway ,,")
	if len(got) != 2 || got[0] != "metrics-server" || got[1] != "gateway" {
		t.Fatalf("got %q", got)
	}
	if len(addonNames("")) != 0 {
		t.Fatal("no names")
	}
}

func TestPrintAddons(t *testing.T) {
	a := addons.Addon{Name: "metrics-server", Description: "The Metrics API."}
	var out bytes.Buffer
	printAddons(&out, []cluster.AddonState{
		{Addon: a, Installed: true, Ready: true, Version: "v0.9.1"},
		{Addon: addons.Addon{Name: "gateway", Description: "Routes."}},
		{Addon: addons.Addon{Name: "other", Description: "x"}, Installed: true, Version: "v1"},
	})
	for _, want := range []string{"metrics-server", "ready, v0.9.1", "gateway", "not installed", "installed, not ready, v1"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, out.String())
		}
	}
}
