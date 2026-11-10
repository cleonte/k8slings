//go:build cluster

package exercises_test

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cleonte/k8slings"
	"github.com/cleonte/k8slings/k8slings/addons"
	"github.com/cleonte/k8slings/k8slings/cluster"
	"github.com/cleonte/k8slings/k8slings/exercises"
)

// K8SLINGS_ADDONS=1 is the add-on job: it installs the add-ons on the proof
// cluster and proves only the exercises that need one. Without it (the shards of
// the cluster job) those exercises are left out, so that a flaky add-on cannot
// fail the basic proofs.
func addonMode() bool { return os.Getenv("K8SLINGS_ADDONS") == "1" }

// needsAddon says whether an exercise requires an add-on (not just topology).
func needsAddon(ex exercises.Exercise) bool {
	for _, r := range ex.Requires {
		if r != "multinode" {
			return true
		}
	}
	return false
}

var addonsOnce struct {
	sync.Once
	err error
}

// coveredHere says whether this run proves ex, and installs the add-ons first
// when it is the add-on run.
func coveredHere(t *testing.T, ctx context.Context, cl *cluster.Cluster, ex exercises.Exercise) bool {
	t.Helper()
	if needsAddon(ex) != addonMode() {
		return false
	}
	if addonMode() {
		installAllAddons(t, ctx, cl)
	}
	return true
}

func installAllAddons(t *testing.T, ctx context.Context, cl *cluster.Cluster) {
	t.Helper()
	addonsOnce.Do(func() {
		var out strings.Builder
		order, err := addons.Order(cl.Registry, addons.Names(cl.Registry))
		if err != nil {
			addonsOnce.err = err
			return
		}
		for _, a := range order {
			files, err := a.Files(k8slings.Addons)
			if err != nil {
				addonsOnce.err = err
				return
			}
			start := time.Now()
			if err := cl.InstallAddon(ctx, a, files, &out); err != nil {
				addonsOnce.err = err
				return
			}
			t.Logf("installed %s %s in %s", a.Name, a.Version, time.Since(start).Round(time.Second))
		}
	})
	if addonsOnce.err != nil {
		t.Fatal(addonsOnce.err)
	}
}

// TestAddonsInstall proves the add-ons on a real cluster: each installs, is
// reported installed and ready, installs a second time without harm, and does
// what it is for.
func TestAddonsInstall(t *testing.T) {
	if !addonMode() {
		t.Skip("the add-on job runs this (K8SLINGS_ADDONS=1)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	cl := proofCluster(t, ctx)
	installAllAddons(t, ctx, cl)

	states, err := cl.AddonStates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, st := range states {
		if !st.Installed || !st.Ready || st.Version != st.Addon.Version {
			t.Errorf("%s: installed=%v ready=%v version=%s, want ready %s", st.Addon.Name, st.Installed, st.Ready, st.Version, st.Addon.Version)
		}
	}

	// A second install is harmless.
	for _, a := range cl.Registry {
		files, _ := a.Files(k8slings.Addons)
		if err := cl.InstallAddon(ctx, a, files, nil); err != nil {
			t.Errorf("installing %s a second time: %v", a.Name, err)
		}
	}

	// The gateway: the GatewayClass is accepted and the Gateway programmed.
	if addons.Find(cl.Registry, "gateway") != nil {
		out, err := cl.Kubectl(ctx, "get", "gatewayclass", "traefik", "-o", `jsonpath={.status.conditions[?(@.type=="Accepted")].status}`)
		if err != nil || strings.TrimSpace(out) != "True" {
			t.Errorf("GatewayClass traefik is not Accepted: %s", out)
		}
		out, err = cl.Kubectl(ctx, "get", "gateway", "gateway", "-n", "addon-traefik", "-o", `jsonpath={.status.conditions[?(@.type=="Programmed")].status}`)
		if err != nil || strings.TrimSpace(out) != "True" {
			t.Errorf("Gateway addon-traefik/gateway is not Programmed: %s", out)
		}
	}

	// metrics-server answers: kubectl top returns rows.
	if addons.Find(cl.Registry, "metrics-server") != nil {
		deadline := time.Now().Add(2 * time.Minute)
		for {
			out, err := cl.Kubectl(ctx, "top", "nodes", "--no-headers")
			if err == nil && strings.TrimSpace(out) != "" {
				t.Logf("kubectl top nodes: %d rows", len(strings.Split(strings.TrimSpace(out), "\n")))
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("kubectl top nodes never returned data: %s", out)
			}
			time.Sleep(3 * time.Second)
		}
	}
}
