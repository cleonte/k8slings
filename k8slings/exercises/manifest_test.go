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
			if namespaceRe.Match(broken) {
				t.Errorf("%s must not set metadata.namespace", ex.Source())
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
