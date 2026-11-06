package exercises_test

import (
	"errors"
	"io/fs"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/cleonte/k8slings"
	"github.com/cleonte/k8slings/k8slings/check"
	"github.com/cleonte/k8slings/k8slings/cluster"
	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/cleonte/k8slings/k8slings/schema"
)

func load(t *testing.T) []exercises.Exercise {
	t.Helper()
	data, err := fs.ReadFile(k8slings.FS, "info.toml")
	if err != nil {
		t.Fatal(err)
	}
	all, err := exercises.ParseInfo(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) == 0 {
		t.Fatal("info.toml lists no exercises")
	}
	return all
}

var (
	markerRe    = regexp.MustCompile(`(?m)^\s*#\s*I\s+AM\s+NOT\s+DONE`)
	todoRe      = regexp.MustCompile(`(?m)#\s*TODO:`)
	namespaceRe = regexp.MustCompile(`(?m)^\s{0,2}namespace:`)
)

func TestManifest(t *testing.T) {
	validator := schema.New(k8slings.Schemas)
	for _, ex := range load(t) {
		t.Run(ex.Name, func(t *testing.T) {
			if !strings.HasPrefix(ex.Path, "exercises/"+ex.Topic+"/") {
				t.Errorf("path %q should be below exercises/%s/", ex.Path, ex.Topic)
			}
			if !strings.HasPrefix(ex.Docs, "https://") {
				t.Errorf("docs %q should be an https link", ex.Docs)
			}
			for _, f := range []string{"README.md", exercises.ExerciseFile, exercises.CheckFile} {
				if _, err := fs.Stat(k8slings.FS, path.Join(ex.Path, f)); err != nil {
					t.Errorf("missing %s: %v", f, err)
				}
			}

			broken, err := fs.ReadFile(k8slings.FS, ex.Source())
			if err != nil {
				t.Fatal(err)
			}
			first, _, _ := strings.Cut(string(broken), "\n")
			if !markerRe.MatchString(first) {
				t.Errorf("the first line of %s must be the 'I AM NOT DONE' marker", ex.Source())
			}
			if !todoRe.Match(broken) {
				t.Errorf("%s needs a '# TODO:' comment at the fix site", ex.Source())
			}
			if ex.Mode != exercises.ModeManifest && namespaceRe.Match(broken) {
				t.Errorf("%s must not set metadata.namespace: k8slings picks the namespace of cluster and fix exercises", ex.Source())
			}

			sol, err := fs.ReadFile(k8slings.Solutions, ex.SolutionPath())
			if err != nil {
				t.Fatalf("missing solution: %v", err)
			}
			if markerRe.Match(sol) || todoRe.Match(sol) {
				t.Errorf("the solution must not keep the marker or the TODO")
			}

			raw, err := fs.ReadFile(k8slings.FS, ex.CheckPath())
			if err != nil {
				t.Fatal(err)
			}
			c, err := check.Load(raw)
			if err != nil {
				t.Fatal(err)
			}
			if c.Schema == "strict" {
				objs, err := exercises.ParseObjects(sol)
				if err != nil {
					t.Fatal(err)
				}
				for _, o := range objs {
					if _, err := validator.Validate(o); err != nil {
						var noSchema schema.ErrNoSchema
						if errors.As(err, &noSchema) {
							t.Errorf("no embedded schema for %s: add it to scripts/gen-schemas.sh", noSchema)
						}
					}
				}
			}
		})
	}
}

// Every folder below exercises/ and solutions/ must belong to an exercise.
func TestNoOrphans(t *testing.T) {
	known := map[string]bool{}
	for _, ex := range load(t) {
		known[ex.Path] = true
	}
	for _, root := range []struct {
		fsys   fs.FS
		prefix string
	}{{k8slings.FS, "exercises"}, {k8slings.Solutions, "solutions"}} {
		err := fs.WalkDir(root.fsys, root.prefix, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || d.Name() == "README.md" {
				return err
			}
			dir := path.Dir(p)
			if root.prefix == "solutions" {
				dir = "exercises/" + strings.TrimPrefix(dir, "solutions/")
			}
			if !known[dir] {
				t.Errorf("%s does not belong to an exercise in info.toml", p)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

var imageRe = regexp.MustCompile(`(?m)^\s*(?:-\s*)?image:\s*["']?([^\s"']+)`)

// Cluster exercises need steps (what to wait for), manifest exercises must not
// have any, and every image a cluster exercise runs is preloaded into the nodes
// so that no exercise depends on a registry.
func TestModesAndImages(t *testing.T) {
	preloaded := map[string]bool{}
	for _, img := range cluster.PreloadImages {
		preloaded[img] = true
	}
	for _, ex := range load(t) {
		raw, err := fs.ReadFile(k8slings.FS, ex.CheckPath())
		if err != nil {
			t.Fatal(err)
		}
		c, err := check.Load(raw)
		if err != nil {
			t.Fatal(err)
		}
		switch ex.Mode {
		case exercises.ModeManifest:
			if c.HasSteps() {
				t.Errorf("%s is a manifest exercise, so its check.yaml must not have steps", ex.Name)
			}
		case exercises.ModeCluster:
			if c.Shipped == "" {
				t.Errorf("%s is a cluster exercise: its check.yaml needs `shipped:`, what the first problem of the unsolved exercise says", ex.Name)
			}
			if !c.HasSteps() {
				t.Errorf("%s is a cluster exercise and needs steps in its check.yaml", ex.Name)
			}
			broken, _ := fs.ReadFile(k8slings.FS, ex.Source())
			sol, _ := fs.ReadFile(k8slings.Solutions, ex.SolutionPath())
			for _, src := range [][]byte{broken, sol} {
				for _, m := range imageRe.FindAllSubmatch(src, -1) {
					if !preloaded[string(m[1])] {
						t.Errorf("%s uses the image %s, which cluster.PreloadImages does not list", ex.Name, m[1])
					}
				}
			}
		}
	}
}
