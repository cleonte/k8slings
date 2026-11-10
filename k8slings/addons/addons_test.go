package addons

import (
	"strings"
	"testing"
	"testing/fstest"
)

func fsys(addonYAML string) fstest.MapFS {
	return fstest.MapFS{
		"addons/a/addon.yaml": {Data: []byte(addonYAML)},
		"addons/a/m.yaml":     {Data: []byte("kind: ServiceAccount\n")},
	}
}

const good = `name: a
version: v1.2.3
source: github.com/x/a
description: d
namespace: addon-a
manifests: [m.yaml]
ready: [deployment/a]
images: [registry.example/a:v1.2.3]
`

func TestLoadAcceptsAGoodEntry(t *testing.T) {
	all, err := Load(fsys(good))
	if err != nil || len(all) != 1 || all[0].Name != "a" {
		t.Fatalf("all=%v err=%v", all, err)
	}
}

func TestLoadRefusesMistakes(t *testing.T) {
	for name, tc := range map[string]struct{ yaml, want string }{
		"owned namespace":        {strings.Replace(good, "addon-a", "k8slings-a", 1), "belongs to the exercises"},
		"no namespace":           {strings.Replace(good, "namespace: addon-a\n", "", 1), "needs a namespace"},
		"name is not the folder": {strings.Replace(good, "name: a", "name: b", 1), "name of its folder"},
		"floating version":       {strings.Replace(good, "v1.2.3\nsource", "latest\nsource", 1), "exact version"},
		"latest image":           {strings.Replace(good, "a:v1.2.3", "a:latest", 1), "pinned"},
		"untagged image":         {strings.Replace(good, "a:v1.2.3", "a", 1), "pinned"},
		"missing manifest":       {strings.Replace(good, "m.yaml", "nope.yaml", 1), "manifest nope.yaml"},
		"nothing to wait for":    {strings.Replace(good, "ready: [deployment/a]\n", "", 1), "something to wait for"},
		"bad ready":              {strings.Replace(good, "deployment/a", "a", 1), "kind/name"},
		"unknown field":          {good + "oops: 1\n", "oops"},
		"unknown requires":       {good + "requires: [zzz]\n", "not in the registry"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Load(fsys(tc.yaml))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestOrderInstallsRequirementsFirst(t *testing.T) {
	all := []Addon{{Name: "gateway", Requires: []string{"gateway-api"}}, {Name: "gateway-api"}, {Name: "metrics-server"}}
	got, err := Order(all, []string{"gateway", "metrics-server", "gateway-api"})
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, a := range got {
		names = append(names, a.Name)
	}
	if strings.Join(names, ",") != "gateway-api,gateway,metrics-server" {
		t.Fatalf("order = %v", names)
	}
	if _, err := Order(all, []string{"zzz"}); err == nil || !strings.Contains(err.Error(), "gateway, gateway-api, metrics-server") {
		t.Fatalf("an unknown name lists the ones there are: %v", err)
	}
	loop := []Addon{{Name: "a", Requires: []string{"b"}}, {Name: "b", Requires: []string{"a"}}}
	if _, err := Order(loop, []string{"a"}); err == nil {
		t.Fatal("a cycle must be refused")
	}
}

func TestMainObject(t *testing.T) {
	a := Addon{Namespace: "kube-system", Ready: []string{"deployment/x"}}
	if ns, obj := a.MainObject(); ns != "kube-system" || obj != "deployment/x" {
		t.Errorf("got %s %s", ns, obj)
	}
	b := Addon{Namespace: "n", Ready: []string{"other/deployment/y"}}
	if ns, obj := b.MainObject(); ns != "other" || obj != "deployment/y" {
		t.Errorf("got %s %s", ns, obj)
	}
	c := Addon{Gateway: "addon-t/gw"}
	if ns, obj := c.MainObject(); ns != "addon-t" || obj != "gateway/gw" {
		t.Errorf("got %s %s", ns, obj)
	}
}
