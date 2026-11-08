package check

import (
	"strings"
	"testing"
)

func TestQueryCheckLoads(t *testing.T) {
	c, err := Load([]byte(`
shipped: "differs"
query:
  sort: true
  command:
    - expr: "verb == 'get' && flags['output'].startsWith('jsonpath=')"
      message: "Use -o jsonpath."
    - expr: "!positional.exists(p, p.startsWith('web-'))"
      message: "Do not name a Pod."
  variant_steps:
    - rollout: deployment/extra
steps:
  - rollout: deployment/web
`))
	if err != nil {
		t.Fatal(err)
	}
	if c.Query.Compare != CompareOutput || !c.HasVariantSteps() || !c.HasSteps() {
		t.Fatalf("%+v", c.Query)
	}
	if got := c.RunCommand("get", []string{"pods"}, map[string]string{"output": "jsonpath={.items}"}); len(got) != 0 {
		t.Errorf("failures = %v", got)
	}
	got := c.RunCommand("get", []string{"web-1"}, map[string]string{"output": "name"})
	if len(got) != 2 || got[0] != "Use -o jsonpath." || got[1] != "Do not name a Pod." {
		t.Errorf("failures = %v", got)
	}
	if got := c.RunCommand("get", nil, map[string]string{}); len(got) != 1 || got[0] != "Use -o jsonpath." {
		t.Errorf("a missing flag counts as a failed assertion, got %v", got)
	}
}

func TestQueryCheckMistakesAreErrors(t *testing.T) {
	for name, tc := range map[string]struct{ yaml, want string }{
		"unknown compare":          {"query:\n  compare: diff\n", "output or manifest"},
		"manifest needs rules":     {"query:\n  compare: manifest\n", "needs objects"},
		"sort with manifest":       {"query:\n  compare: manifest\n  sort: true\nobjects:\n  - select: {kind: Deployment}\n    assert: []\n", "sort"},
		"objects with output":      {"query:\n  compare: output\nobjects:\n  - select: {kind: Deployment}\n    assert: []\n", "reference answer"},
		"command without message":  {"query:\n  command:\n    - expr: \"verb == 'get'\"\n      message: \"\"\n", "needs a message"},
		"command not boolean":      {"query:\n  command:\n    - expr: \"verb\"\n      message: m\n", "boolean"},
		"command does not compile": {"query:\n  command:\n    - expr: \"nope == 1\"\n      message: m\n", "query.command[0]"},
		"bad variant step":         {"query:\n  variant_steps:\n    - rollout: web\n", "query.variant_steps[0]"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Load([]byte(tc.yaml))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
}
