package schema_test

import (
	"strings"
	"testing"

	"github.com/cleonte/k8slings"
	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/cleonte/k8slings/k8slings/schema"
)

func problems(t *testing.T, yaml string) []string {
	t.Helper()
	objs, err := exercises.ParseObjects([]byte(yaml))
	if err != nil || len(objs) != 1 {
		t.Fatalf("%v", err)
	}
	p, err := schema.New(k8slings.Schemas).Validate(objs[0])
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// HTTPRoute, Gateway, GatewayClass and ReferenceGrant validate offline, from
// schemas converted from the pinned CRDs, as strictly as the core kinds.
func TestGatewayAPIKindsValidateOffline(t *testing.T) {
	route := `apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: api
  labels: {a: b}
spec:
  parentRefs:
    - name: gateway
      namespace: addon-traefik
      sectionName: web
  hostnames: [shop.example.test]
  rules:
    - matches:
        - path: {type: PathPrefix, value: /api}
          headers: [{name: X-Team, value: blue}]
      backendRefs:
        - {name: v1, port: 80, weight: 90}
        - {name: v2, port: 80, weight: 10}
`
	if p := problems(t, route); len(p) > 0 {
		t.Fatalf("a good HTTPRoute: %v", p)
	}
	for name, tc := range map[string]struct{ from, to, want string }{
		"a misspelled parentRef field": {"sectionName: web", "sectionNmae: web", "sectionNmae"},
		"a misspelled match field":     {"path: {type: PathPrefix", "pth: {type: PathPrefix", "pth"},
		"a weight that is a string":    {"weight: 90", "weight: ninety", "weight"},
		"an unknown path type":         {"type: PathPrefix", "type: Prefix", "type"},
		"a field of the wrong level":   {"  hostnames: [shop.example.test]", "  hostname: shop.example.test", "hostname"},
	} {
		t.Run(name, func(t *testing.T) {
			bad := strings.Replace(route, tc.from, tc.to, 1)
			p := problems(t, bad)
			if len(p) == 0 || !strings.Contains(strings.Join(p, "\n"), tc.want) {
				t.Fatalf("problems = %v, want one about %q", p, tc.want)
			}
		})
	}

	gateway := `apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata: {name: shop}
spec:
  gatewayClassName: traefik
  listeners:
    - name: web
      port: 8000
      protocol: HTTP
      hostname: shop.example.test
      allowedRoutes: {namespaces: {from: Same}}
`
	if p := problems(t, gateway); len(p) > 0 {
		t.Fatalf("a good Gateway: %v", p)
	}
	if p := problems(t, strings.Replace(gateway, "allowedRoutes:", "allowedRoute:", 1)); len(p) == 0 {
		t.Error("a misspelled allowedRoutes was accepted")
	}
	if p := problems(t, "apiVersion: gateway.networking.k8s.io/v1\nkind: GatewayClass\nmetadata: {name: x}\nspec: {controllerName: example.com/controller}\n"); len(p) > 0 {
		t.Errorf("a good GatewayClass: %v", p)
	}
	if p := problems(t, "apiVersion: gateway.networking.k8s.io/v1\nkind: ReferenceGrant\nmetadata: {name: x}\nspec:\n  from: [{group: gateway.networking.k8s.io, kind: HTTPRoute, namespace: a}]\n  to: [{group: \"\", kind: Service}]\n"); len(p) > 0 {
		t.Errorf("a good ReferenceGrant: %v", p)
	}
}

// What the Traefik add-on creates validates against the same schemas: a check
// that the converted schemas accept what a real controller manifest says.
func TestTheGatewayAddOnObjectsValidate(t *testing.T) {
	data, err := k8slings.Addons.ReadFile("addons/gateway/traefik.yaml")
	if err != nil {
		t.Fatal(err)
	}
	objs, err := exercises.ParseObjects(data)
	if err != nil {
		t.Fatal(err)
	}
	v := schema.New(k8slings.Schemas)
	n := 0
	for _, o := range objs {
		if o["kind"] != "Gateway" && o["kind"] != "GatewayClass" {
			continue
		}
		n++
		if p, err := v.Validate(o); err != nil || len(p) > 0 {
			t.Errorf("%v: err=%v problems=%v", o["kind"], err, p)
		}
	}
	if n != 2 {
		t.Fatalf("checked %d objects, want a Gateway and a GatewayClass", n)
	}
}
