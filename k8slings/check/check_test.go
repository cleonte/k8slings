package check

import (
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

const manifest = `
apiVersion: apps/v1
kind: Deployment
metadata: {name: web}
spec:
  replicas: 3
  selector: {matchLabels: {app: web}}
  template:
    metadata: {labels: {app: web}}
    spec:
      containers:
        - name: nginx
          image: nginx:1.27
          readinessProbe: {httpGet: {path: /, port: 80}}
---
apiVersion: v1
kind: Service
metadata: {name: web}
spec:
  selector: {app: web}
  ports: [{port: 80}]
`

func objects(t *testing.T, s string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, d := range strings.Split(s, "\n---\n") {
		var m map[string]any
		if err := yaml.Unmarshal([]byte(d), &m); err != nil {
			t.Fatal(err)
		}
		out = append(out, m)
	}
	return out
}

const checkYAML = `
objects:
  - select: {kind: Deployment}
    assert:
      - expr: object.spec.replicas == 3
        message: three replicas
      - expr: object.spec.selector.matchLabels.all(k, object.spec.template.metadata.labels[k] == object.spec.selector.matchLabels[k])
        message: selector must match labels
      - expr: object.spec.template.spec.containers.all(c, has(c.readinessProbe))
        message: readiness probe needed
  - select: {kind: Service}
    assert:
      - expr: objects.exists(d, d.kind == 'Deployment' && object.spec.selector.all(k, d.spec.template.metadata.labels[k] == object.spec.selector[k]))
        message: service matches no deployment
`

func TestRunPasses(t *testing.T) {
	c, err := Load([]byte(checkYAML))
	if err != nil {
		t.Fatal(err)
	}
	if f := c.Run(objects(t, manifest)); len(f) != 0 {
		t.Fatalf("unexpected failures: %v", f)
	}
}

func TestRunReportsMessages(t *testing.T) {
	c, _ := Load([]byte(checkYAML))
	broken := strings.NewReplacer("replicas: 3", "replicas: 2", "readinessProbe: {httpGet: {path: /, port: 80}}", "ports: [{containerPort: 80}]").Replace(manifest)
	f := c.Run(objects(t, broken))
	want := []string{"three replicas", "readiness probe needed"}
	if strings.Join(f, "|") != strings.Join(want, "|") {
		t.Fatalf("got %v, want %v", f, want)
	}
}

func TestRunMissingFieldIsAFailureNotAPanic(t *testing.T) {
	c, _ := Load([]byte(checkYAML))
	f := c.Run(objects(t, strings.Replace(manifest, "replicas: 3", "", 1)))
	if len(f) != 1 || f[0] != "three replicas" {
		t.Fatalf("got %v", f)
	}
}

func TestRunSelectCounts(t *testing.T) {
	c, _ := Load([]byte(checkYAML))
	if f := c.Run(objects(t, strings.Split(manifest, "\n---\n")[1])); len(f) < 2 {
		t.Fatalf("expected no-Deployment and service failures, got %v", f)
	}
}

func TestLoadRejectsAuthoringMistakes(t *testing.T) {
	for name, in := range map[string]string{
		"no message":  "objects:\n- select: {kind: Pod}\n  assert:\n  - expr: 'true'\n",
		"bad expr":    "objects:\n- select: {kind: Pod}\n  assert:\n  - {expr: 'object.spec.==', message: m}\n",
		"not bool":    "objects:\n- select: {kind: Pod}\n  assert:\n  - {expr: '1 + 1', message: m}\n",
		"no kind":     "objects:\n- assert:\n  - {expr: 'true', message: m}\n",
		"unknown key": "objcts: []\n",
		"bad schema":  "schema: lenient\n",
	} {
		if _, err := Load([]byte(in)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestQuantityFunction(t *testing.T) {
	c, err := Load([]byte(`
schema: "off"
objects:
  - select: {kind: Pod}
    assert:
      - {expr: "quantity('64Mi') == 67108864.0", message: "binary suffix"}
      - {expr: "quantity('500m') < quantity('1')", message: "milli-cores"}
      - {expr: "quantity('9Mi') < quantity('10Mi')", message: "numbers, not strings"}
      - {expr: "quantity('1Gi') > quantity('1G')", message: "binary is bigger than decimal"}
      - {expr: "quantity('64Gi') > quantity('64Mi')", message: "the broken exercise asked for far too much"}
      - {expr: "quantity(object.spec.containers[0].resources.limits.memory) <= quantity('1Gi')", message: "reads a value from the object"}
      - {expr: "quantity('2') == 2.0 && quantity('1500m') == 1.5 && quantity('1e3') == 1000.0", message: "plain, fractional and exponent"}
`))
	if err != nil {
		t.Fatal(err)
	}
	pod := `{"kind":"Pod","spec":{"containers":[{"resources":{"limits":{"memory":"256Mi"}}}]}}`
	var obj map[string]any
	if err := yaml.Unmarshal([]byte(pod), &obj); err != nil {
		t.Fatal(err)
	}
	if f := c.Run([]map[string]any{obj}); len(f) != 0 {
		t.Fatalf("failures: %v", f)
	}
}

func TestQuantityOfNonsenseIsAFailedAssertion(t *testing.T) {
	c, err := Load([]byte("schema: \"off\"\nobjects:\n  - select: {kind: Pod}\n    assert:\n      - {expr: \"quantity('lots') > quantity('1')\", message: \"not a quantity\"}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if f := c.Run([]map[string]any{{"kind": "Pod"}}); len(f) != 1 || f[0] != "not a quantity" {
		t.Fatalf("got %v", f)
	}
}

func TestParseQuantity(t *testing.T) {
	for in, want := range map[string]float64{"1": 1, "100m": 0.1, "2k": 2000, "1Ki": 1024, "1.5Gi": 1.5 * (1 << 30), "0": 0, "3e2": 300} {
		got, err := ParseQuantity(in)
		if err != nil || got != want {
			t.Errorf("ParseQuantity(%q) = %v, %v, want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "Mi", "12XB", "abc"} {
		if _, err := ParseQuantity(in); err == nil {
			t.Errorf("ParseQuantity(%q) should fail", in)
		}
	}
}
