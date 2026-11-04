package schema

import (
	"os"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

func parse(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := yaml.Unmarshal([]byte(s), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func validator() *Validator { return New(os.DirFS("../..")) }

const deployment = `
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
`

func TestFileName(t *testing.T) {
	for _, tc := range []struct{ apiVersion, kind, want string }{
		{"v1", "Pod", "pod-v1"},
		{"apps/v1", "Deployment", "deployment-apps-v1"},
		{"networking.k8s.io/v1", "NetworkPolicy", "networkpolicy-networking-v1"},
		{"rbac.authorization.k8s.io/v1", "RoleBinding", "rolebinding-rbac-v1"},
		{"autoscaling/v2", "HorizontalPodAutoscaler", "horizontalpodautoscaler-autoscaling-v2"},
	} {
		if got := FileName(tc.apiVersion, tc.kind); got != tc.want {
			t.Errorf("FileName(%q, %q) = %q, want %q", tc.apiVersion, tc.kind, got, tc.want)
		}
	}
}

func TestValidateAcceptsValidObject(t *testing.T) {
	problems, err := validator().Validate(parse(t, deployment))
	if err != nil || len(problems) != 0 {
		t.Fatalf("problems=%v err=%v", problems, err)
	}
}

func TestValidateRejectsMisspelledField(t *testing.T) {
	problems, err := validator().Validate(parse(t, strings.Replace(deployment, "containers:", "contianers:", 1)))
	if err != nil {
		t.Fatal(err)
	}
	all := strings.Join(problems, "\n")
	if !strings.Contains(all, "spec.template.spec") || !strings.Contains(all, "contianers") {
		t.Fatalf("expected a problem naming spec.template.spec and contianers, got:\n%s", all)
	}
	if strings.Contains(all, "k8s://") || strings.Contains(all, "jsonschema validation failed") {
		t.Fatalf("problems leak validator internals:\n%s", all)
	}
}

func TestValidateRejectsWrongType(t *testing.T) {
	problems, _ := validator().Validate(parse(t, strings.Replace(deployment, "replicas: 3", "replicas: three", 1)))
	if len(problems) == 0 || !strings.Contains(problems[0], "spec.replicas") {
		t.Fatalf("expected a spec.replicas problem, got %v", problems)
	}
}

func TestValidateUnknownKind(t *testing.T) {
	_, err := validator().Validate(parse(t, "apiVersion: example.com/v1\nkind: Widget\n"))
	if _, ok := err.(ErrNoSchema); !ok {
		t.Fatalf("want ErrNoSchema, got %v", err)
	}
}
