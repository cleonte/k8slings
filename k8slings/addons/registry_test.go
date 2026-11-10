package addons_test

import (
	"context"
	"io/fs"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/cleonte/k8slings"
	"github.com/cleonte/k8slings/k8slings/addons"
	"github.com/cleonte/k8slings/k8slings/cluster"
	"github.com/cleonte/k8slings/k8slings/exercises"
)

func registry(t *testing.T) []addons.Addon {
	t.Helper()
	all, err := addons.Load(k8slings.Addons)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) == 0 {
		t.Fatal("the registry is empty")
	}
	return all
}

// The prefix the registry refuses and the prefix exercise paths accept must be
// the same, or an add-on could slip between them.
func TestTheRegistryAndTheClusterAgreeOnWhatBelongsToExercises(t *testing.T) {
	if addons.OwnedPrefix != cluster.NamespacePrefix {
		t.Fatalf("addons.OwnedPrefix = %q, cluster.NamespacePrefix = %q", addons.OwnedPrefix, cluster.NamespacePrefix)
	}
}

// clusterScoped are the kinds whose objects are not in a namespace: what an
// add-on changes cluster-wide. ClusterWide has to list every one of them.
var clusterScoped = []string{
	"Namespace", "ClusterRole", "ClusterRoleBinding", "APIService", "CustomResourceDefinition",
	"GatewayClass", "IngressClass", "StorageClass", "PriorityClass",
	"ValidatingAdmissionPolicy", "ValidatingAdmissionPolicyBinding",
	"MutatingWebhookConfiguration", "ValidatingWebhookConfiguration",
}

var imageRe = regexp.MustCompile(`(?m)^\s*(?:-\s*)?image:\s*["']?([^\s"']+)`)

func TestEveryAddOnIsPinnedAndDeclaresWhatItChangesClusterWide(t *testing.T) {
	for _, a := range registry(t) {
		t.Run(a.Name, func(t *testing.T) {
			files, err := a.Files(k8slings.Addons)
			if err != nil {
				t.Fatal(err)
			}
			found := map[string]bool{}
			var images []string
			for i, f := range files {
				objs, err := exercises.ParseObjects(f)
				if err != nil || len(objs) == 0 {
					t.Fatalf("%s does not parse, or has no objects: %v", a.Manifests[i], err)
				}
				for _, o := range objs {
					kind, _ := o["kind"].(string)
					md, _ := o["metadata"].(map[string]any)
					name, _ := md["name"].(string)
					if slices.Contains(clusterScoped, kind) {
						found[kind+" "+name] = true
					}
					if ns, _ := md["namespace"].(string); strings.HasPrefix(ns, addons.OwnedPrefix) {
						t.Errorf("%s %s is in %s, which an exercise could reach", kind, name, ns)
					}
				}
				for _, m := range imageRe.FindAllSubmatch(f, -1) {
					images = append(images, string(m[1]))
				}
			}
			for obj := range found {
				if !slices.Contains(a.ClusterWide, obj) {
					t.Errorf("the manifests create the cluster-wide object %q, which clusterWide does not list", obj)
				}
			}
			for _, obj := range a.ClusterWide {
				if !found[obj] {
					t.Errorf("clusterWide lists %q, which the manifests do not create", obj)
				}
			}
			for _, img := range images {
				if !slices.Contains(a.Images, img) {
					t.Errorf("the manifests run the image %s, which images does not list (it would be pulled at exercise time)", img)
				}
			}
			for _, img := range a.Images {
				if !slices.Contains(images, img) && len(images) > 0 {
					t.Errorf("images lists %s, which the manifests do not use", img)
				}
			}
		})
	}
}

// fake is a Runner that counts what is run.
type fake struct{ calls int }

func (f *fake) Has(string) bool { return true }
func (f *fake) Run(context.Context, cluster.Cmd) error {
	f.calls++
	return nil
}

// Every path an exercise takes into a namespace refuses the namespace of every
// add-on, and runs nothing: a reset, a seeding or a cleanup of an exercise can
// never touch an add-on.
func TestNoExercisePathReachesAnAddOnNamespace(t *testing.T) {
	ctx := context.Background()
	for _, a := range registry(t) {
		nss := []string{a.Namespace}
		if ns, _ := a.MainObject(); ns != a.Namespace {
			nss = append(nss, ns)
		}
		for _, ns := range nss {
			f := &fake{}
			c := &cluster.Cluster{Name: "k8slings", Kubeconfig: "/x", Runner: f}
			paths := map[string]error{
				"ResetNamespace":    c.ResetNamespace(ctx, ns),
				"ReleaseFinalizers": c.ReleaseFinalizers(ctx, ns),
				"Seed":              c.Seed(ctx, ns, "x", [][]byte{[]byte("kind: ConfigMap\n")}),
				"Delete":            c.Delete(ctx, ns, []byte("kind: ConfigMap\n")),
			}
			for name, err := range paths {
				if err == nil {
					t.Errorf("%s accepted the namespace %s of the add-on %s", name, ns, a.Name)
				}
			}
			if f.calls != 0 {
				t.Errorf("%s: %d programs were run for the namespace %s", a.Name, f.calls, ns)
			}
		}
	}
}

// A requires entry of an exercise that is not multinode must name an add-on.
func TestExercisesRequireAddOnsThatExist(t *testing.T) {
	all := registry(t)
	data, err := fs.ReadFile(k8slings.FS, "info.toml")
	if err != nil {
		t.Fatal(err)
	}
	list, err := exercises.ParseInfo(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, ex := range list {
		for _, need := range ex.Requires {
			if need != "multinode" && addons.Find(all, need) == nil {
				t.Errorf("%s requires %q, which is neither multinode nor an add-on (they are: %s)", ex.Name, need, strings.Join(addons.Names(all), ", "))
			}
		}
		if len(ex.Requires) > 0 && ex.Mode == exercises.ModeManifest {
			t.Errorf("%s is a manifest exercise and cannot require %v: it runs offline", ex.Name, ex.Requires)
		}
	}
}
