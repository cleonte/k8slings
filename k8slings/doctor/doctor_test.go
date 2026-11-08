package doctor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/cleonte/k8slings/k8slings/exercises"
	ws "github.com/cleonte/k8slings/k8slings/workspace"
)

var schemas = fstest.MapFS{
	"schemas/VERSION":         {Data: []byte("v1.37.0\n")},
	"schemas/pod-v1.json.gz":  {},
	"schemas/node-v1.json.gz": {},
}

const info = "[[exercises]]\nname='a'\npath='exercises/t/a'\ntopic='t'\nmode='manifest'\nhint='h'\n"

func workspace(t *testing.T, manifest string) string {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.WriteFile("info.toml", []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	return "info.toml"
}

func byName(checks []Check, name string) Check {
	for _, c := range checks {
		if c.Name == name {
			return c
		}
	}
	return Check{Name: "(missing) " + name}
}

func TestHealthyWorkspace(t *testing.T) {
	infoFile := workspace(t, info)
	if err := os.MkdirAll(filepath.Join("exercises", "t", "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("exercises", "t", "a", "exercise.yaml"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	checks := Run(context.Background(), infoFile, DefaultEnv(schemas, nil))
	if Failed(checks) {
		t.Fatalf("unexpected failure: %+v", checks)
	}
	if d := byName(checks, "Kubernetes schemas").Detail; !strings.Contains(d, "v1.37.0") || !strings.Contains(d, "2 kinds") {
		t.Errorf("schema detail = %q", d)
	}
	if c := byName(checks, "Exercise modes"); c.Status != OK || c.Group != Offline {
		t.Errorf("a manifest-only workspace should be OK: %+v", byName(checks, "Exercise modes"))
	}
}

func TestMissingExerciseFile(t *testing.T) {
	infoFile := workspace(t, info)
	checks := Run(context.Background(), infoFile, DefaultEnv(schemas, nil))
	c := byName(checks, "Exercise files")
	if c.Status != Fail || !strings.Contains(c.Detail, "reset") {
		t.Fatalf("got %+v", c)
	}
	if !Failed(checks) {
		t.Fatal("a missing exercise file is a failure")
	}
}

func TestNoManifest(t *testing.T) {
	t.Chdir(t.TempDir())
	checks := Run(context.Background(), "info.toml", DefaultEnv(schemas, nil))
	c := byName(checks, "Exercise manifest")
	if c.Status != Fail || !strings.Contains(c.Detail, "k8slings init") {
		t.Fatalf("got %+v", c)
	}
}

func TestClusterExercisesAreAWarning(t *testing.T) {
	infoFile := workspace(t, info+"[[exercises]]\nname='b'\npath='exercises/t/b'\ntopic='t'\nmode='cluster'\nhint='h'\n")
	c := byName(Run(context.Background(), infoFile, Env{FileExists: func(string) bool { return true }, Schemas: schemas}), "Exercise modes")
	if c.Status != Warn || !strings.Contains(c.Detail, "1 of 2") {
		t.Fatalf("got %+v", c)
	}
}

func TestMissingSchemas(t *testing.T) {
	infoFile := workspace(t, info)
	checks := Run(context.Background(), infoFile, Env{FileExists: func(string) bool { return true }, Schemas: fstest.MapFS{}})
	if byName(checks, "Kubernetes schemas").Status != Fail {
		t.Fatalf("got %+v", byName(checks, "Kubernetes schemas"))
	}
}

// A workspace made by init has no setup steps, and doctor must not look for them.
func TestDoctorNeverMissesSetupFilesInTheWorkspace(t *testing.T) {
	src := fstest.MapFS{
		"info.toml":                  {Data: []byte("[[exercises]]\nname='f'\npath='exercises/t/f'\ntopic='t'\nmode='fix'\nhint='h'\n")},
		"exercises/t/f/README.md":    {Data: []byte("# ticket\n")},
		"exercises/t/f/setup.yaml":   {Data: []byte("one\n")},
		"exercises/t/f/setup-2.yaml": {Data: []byte("two\n")},
		"exercises/t/f/check.yaml":   {Data: []byte("steps: []\n")},
	}
	dest := filepath.Join(t.TempDir(), "ws")
	if err := ws.Init(src, dest); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dest)
	all, err := exercises.List("info.toml")
	if err != nil {
		t.Fatal(err)
	}
	env := Env{FileExists: func(p string) bool { _, err := os.Stat(p); return err == nil }}
	if c := filesCheck(all, env); c.Status != OK {
		t.Fatalf("a fresh workspace must pass: %v %s", c.Status, c.Detail)
	}
}

func TestDoctorWarnsWhenTheFolderLacksExercisesOfThisVersion(t *testing.T) {
	infoFile := workspace(t, info)
	find := func(env Env) *Check {
		for _, c := range Run(context.Background(), infoFile, env) {
			if c.Name == "Exercise set" {
				return &c
			}
		}
		return nil
	}
	env := DefaultEnv(schemas, nil)
	if find(env) != nil {
		t.Fatal("without a way to know, there is no check")
	}
	env.Missing = func() []string { return []string{"x1", "x2"} }
	if c := find(env); c == nil || c.Status != Warn || !strings.Contains(c.Detail, "k8slings update") || !strings.Contains(c.Detail, "x1, x2") {
		t.Fatalf("got %+v", c)
	}
	env.Missing = func() []string { return nil }
	if c := find(env); c == nil || c.Status != OK {
		t.Fatalf("got %+v", c)
	}
}

func TestDoctorSaysWhetherKubectlIsThereForTheOfflineQueryExercises(t *testing.T) {
	infoFile := workspace(t, info+"[[exercises]]\nname='gen'\npath='exercises/t/gen'\ntopic='t'\nmode='query'\noffline=true\nhint='h'\n")
	for has, want := range map[bool]Status{true: OK, false: Warn} {
		env := Env{FileExists: func(string) bool { return true }, Schemas: schemas, HasKubectl: func() bool { return has }}
		c := byName(Run(context.Background(), infoFile, env), "kubectl")
		if c.Status != want || c.Group != Offline || !strings.Contains(c.Detail, "1 query exercise(s)") {
			t.Errorf("kubectl present=%v: got %+v", has, c)
		}
	}
	// Without such an exercise there is nothing to say about it.
	plain := workspace(t, info)
	env := Env{FileExists: func(string) bool { return true }, Schemas: schemas, HasKubectl: func() bool { return false }}
	for _, c := range Run(context.Background(), plain, env) {
		if c.Name == "kubectl" {
			t.Errorf("unexpected %+v", c)
		}
	}
}
