package exercises_test

import (
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/cleonte/k8slings"
	"github.com/cleonte/k8slings/k8slings/check"
	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/cleonte/k8slings/k8slings/query"
	"github.com/cleonte/k8slings/k8slings/schema"
)

// variantMessage is what a failed variant says, which a shortcut that hard-codes
// today's names is refused with.
const variantMessage = "other objects are in the namespace"

// A query exercise has the learner write one kubectl command in answer.kubectl.
// This is everything that can be checked offline; TestQuerySolutions (cluster
// tag) proves the rest against a real cluster.
func testQueryExercise(t *testing.T, ex exercises.Exercise) {
	t.Helper()
	for _, f := range []string{exercises.ReadmeFile, exercises.AnswerFile, exercises.CheckFile} {
		if _, err := fs.Stat(k8slings.FS, path.Join(ex.Path, f)); err != nil {
			t.Errorf("missing %s: %v", f, err)
		}
	}
	if _, err := fs.Stat(k8slings.FS, path.Join(ex.Path, exercises.ExerciseFile)); err == nil {
		t.Errorf("a query exercise has no %s: the learner writes %s", exercises.ExerciseFile, exercises.AnswerFile)
	}

	raw, err := fs.ReadFile(k8slings.FS, ex.CheckPath())
	if err != nil {
		t.Fatal(err)
	}
	c, err := check.Load(raw)
	if err != nil {
		t.Fatal(err)
	}
	if c.Query == nil {
		t.Fatalf("%s: check.yaml needs a query section", ex.Name)
	}
	if c.Shipped == "" {
		t.Errorf("%s: check.yaml needs `shipped:`, what the first problem of the shipped answer says", ex.Name)
	}

	// The file the learner gets.
	placeholder, err := fs.ReadFile(k8slings.FS, ex.Source())
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(placeholder), "\n")
	if !markerRe.MatchString(lines[0]) {
		t.Errorf("the first line of %s must be the 'I AM NOT DONE' marker", ex.Source())
	}
	if len(lines) < 2 || !strings.HasPrefix(lines[1], "# k8slings reads this file") || !strings.Contains(lines[1], "not a script") {
		t.Errorf("line 2 of %s must say: # k8slings reads this file; it is not a script, do not run it with a shell", ex.Source())
	}
	if !todoRe.Match(placeholder) {
		t.Errorf("%s needs a '# TODO:' comment", ex.Source())
	}

	opt := query.Options{AllNamespaces: c.Query.AllNamespaces}
	if !ex.Offline {
		opt.Namespace = ex.Namespace()
	}
	if _, err := query.Parse(string(placeholder), opt); err != nil {
		t.Errorf("the shipped answer must be a command that runs, and prints the wrong thing: %v", err)
	}
	sol, err := fs.ReadFile(k8slings.Solutions, ex.SolutionPath())
	if err != nil {
		t.Fatalf("missing reference answer: %v", err)
	}
	if markerRe.Match(sol) || todoRe.Match(sol) {
		t.Error("the reference answer must not keep the marker or the TODO")
	}
	cmd, err := query.Parse(string(sol), opt)
	if err != nil {
		t.Fatalf("the reference answer is refused: %v", err)
	}
	if failures := c.RunCommand(cmd.Verb, cmd.Positional, cmd.Flags); len(failures) > 0 {
		t.Errorf("the reference answer fails the command assertions of its own check: %v", failures)
	}
	if ex.Offline && !cmd.Offline {
		t.Error("an offline exercise needs a reference answer that works without a cluster (create ... --dry-run=client)")
	}

	// Offline means no cluster at all, so no state; an exercise that uses the
	// cluster may have none either (kubectl explain asks the API server only).
	_, setupErr := fs.Stat(k8slings.FS, path.Join(ex.Path, exercises.SetupFile))
	hasSetup := setupErr == nil
	if ex.Offline && hasSetup {
		t.Errorf("%s is offline, so it has no cluster state to set up", ex.Name)
	}
	var setupObjs []map[string]any
	if hasSetup {
		files, err := exercises.SetupFiles(k8slings.FS, ex.Path)
		if err != nil {
			t.Fatal(err)
		}
		setupObjs = validateSetup(t, files)
	}

	// The variant only adds.
	variant, err := fs.ReadFile(k8slings.FS, path.Join(ex.Path, exercises.VariantFile))
	hasVariant := err == nil
	if hasVariant {
		if !hasSetup {
			t.Error("a variant adds objects to the state: without a setup.yaml there is none")
		}
		if c.Query.Compare == check.CompareManifest {
			t.Error("a variant is for output comparisons")
		}
		if namespaceRe.Match(variant) {
			t.Errorf("%s must not set metadata.namespace", exercises.VariantFile)
		}
		objs, err := exercises.ParseObjects(variant)
		if err != nil || len(objs) == 0 {
			t.Fatalf("%s does not parse, or has no objects: %v", exercises.VariantFile, err)
		}
		have := map[string]bool{}
		for _, o := range setupObjs {
			have[objectKey(o)] = true
		}
		needsWait := false
		for _, o := range objs {
			if have[objectKey(o)] {
				t.Errorf("%s changes %s, which the state has: a variant only adds objects, so that taking them away restores the state exactly", exercises.VariantFile, objectKey(o))
			}
			switch o["kind"] {
			case "Pod":
				needsWait = true
				spec, _ := o["spec"].(map[string]any)
				if g, ok := spec["terminationGracePeriodSeconds"].(float64); !ok || g > 1 {
					t.Errorf("%s: Pod %s needs terminationGracePeriodSeconds of 0 or 1, so that it is gone when the variant is taken away", exercises.VariantFile, objectName(o))
				}
			case "Deployment", "StatefulSet", "DaemonSet", "Job", "ReplicaSet":
				spec, _ := o["spec"].(map[string]any)
				if r, ok := spec["replicas"].(float64); o["kind"] == "Job" || o["kind"] == "DaemonSet" || !ok || r != 0 {
					t.Errorf("%s: %s adds Pods that cannot be taken away at once; use bare Pods, or replicas: 0", exercises.VariantFile, objectKey(o))
				}
			}
		}
		if needsWait && len(c.Query.VariantSteps) == 0 {
			t.Error("the variant adds Pods: check.yaml needs query.variant_steps, what to wait for before the command is run again")
		}
	} else if len(c.Query.VariantSteps) > 0 {
		t.Error("query.variant_steps without a variant")
	}

	// The shortcuts are hard-coded or cheating answers that must be refused.
	shortcuts, _ := fs.Glob(k8slings.Solutions, path.Join(path.Dir(ex.SolutionPath()), "shortcut-*.kubectl"))
	if len(shortcuts) == 0 {
		t.Errorf("%s needs at least one shortcut-*.kubectl: an answer that cheats, which the check must refuse", ex.Name)
	}
	for _, f := range shortcuts {
		data, err := fs.ReadFile(k8slings.Solutions, f)
		if err != nil {
			t.Fatal(err)
		}
		first, _, _ := strings.Cut(string(data), "\n")
		m := rejectedRe.FindStringSubmatch(first)
		if m == nil {
			t.Errorf(`%s must start with: # rejected: "what the check says"`, f)
			continue
		}
		if _, err := query.Parse(string(data), opt); err != nil {
			t.Errorf("%s must be a command that is accepted and then refused by the check, not by the parser: %v", f, err)
		}
		declared := hasVariantOrMessage(c, hasVariant, m[1])
		if !declared {
			t.Errorf("%s says it is refused with %q, which is neither a message of query.command nor the variant's (%q)", f, m[1], variantMessage)
		}
	}
}

func hasVariantOrMessage(c *check.Compiled, hasVariant bool, text string) bool {
	if hasVariant && strings.Contains(variantMessage, text) {
		return true
	}
	for _, a := range c.Query.Command {
		if strings.Contains(a.Message, text) {
			return true
		}
	}
	for _, r := range c.Objects { // the assertions on the output of a manifest comparison
		for _, a := range r.Assert {
			if strings.Contains(a.Message, text) {
				return true
			}
		}
	}
	return false
}

func objectKey(o map[string]any) string {
	kind, _ := o["kind"].(string)
	return kind + "/" + objectName(o)
}

func objectName(o map[string]any) string {
	md, _ := o["metadata"].(map[string]any)
	n, _ := md["name"].(string)
	return n
}

// validateSetup checks the setup files of a stateful exercise as for a fix
// exercise (they parse, never set a namespace, have a schema) and returns the
// objects of all steps.
func validateSetup(t *testing.T, files []string) []map[string]any {
	t.Helper()
	validator := schema.New(k8slings.Schemas)
	var all []map[string]any
	for _, f := range files {
		setup, err := fs.ReadFile(k8slings.FS, f)
		if err != nil {
			t.Fatal(err)
		}
		if namespaceRe.Match(setup) {
			t.Errorf("%s must not set metadata.namespace: k8slings picks it", f)
		}
		objs, err := exercises.ParseObjects(setup)
		if err != nil || len(objs) == 0 {
			t.Fatalf("%s does not parse, or has no objects: %v", f, err)
		}
		for _, o := range objs {
			if _, err := validator.Validate(o); err != nil {
				if strings.Contains(err.Error(), "no schema embedded") {
					t.Errorf("no embedded schema for %s %s: add it to scripts/gen-schemas.sh", o["kind"], objectName(o))
				}
			}
		}
		all = append(all, objs...)
	}
	return all
}

// What an older kubectl prints for create --dry-run=client -o yaml has nulls in
// it (creationTimestamp: null, status: {}, strategy: {}, resources: {}). The
// strict schema has to accept that: an offline learner may have any kubectl.
func TestGeneratedYAMLOfAnOlderKubectlPassesTheSchema(t *testing.T) {
	validator := schema.New(k8slings.Schemas)
	for name, src := range map[string]string{
		"deployment": `apiVersion: apps/v1
kind: Deployment
metadata:
  creationTimestamp: null
  labels:
    app: web
  name: web
spec:
  replicas: 3
  selector:
    matchLabels:
      app: web
  strategy: {}
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: web
    spec:
      containers:
      - image: nginx:1.27
        name: nginx
        resources: {}
status: {}
`,
		"service": `apiVersion: v1
kind: Service
metadata:
  creationTimestamp: null
  labels:
    app: web
  name: web
spec:
  ports:
  - name: 80-8080
    port: 80
    protocol: TCP
    targetPort: 8080
  selector:
    app: web
  type: ClusterIP
status:
  loadBalancer: {}
`,
	} {
		objs, err := exercises.ParseObjects([]byte(src))
		if err != nil || len(objs) != 1 {
			t.Fatalf("%s: %v", name, err)
		}
		problems, err := validator.Validate(objs[0])
		if err != nil || len(problems) > 0 {
			t.Errorf("%s: err=%v problems=%v", name, err, problems)
		}
	}
}
