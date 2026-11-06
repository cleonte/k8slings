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
			if ex.Mode == exercises.ModeFix {
				testFixExercise(t, ex)
				return
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

// imageRe finds the images in a manifest (`image: x`) and in a kubectl script
// (`--image=x`, `set image deployment/web app=x`).
var imageRe = regexp.MustCompile(`(?m)(?:^\s*(?:-\s*)?image:\s*["']?|--image=|set image \S+ \S+=)([^\s"']+)`)

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
		case exercises.ModeCluster, exercises.ModeFix:
			if c.Shipped == "" {
				t.Errorf("%s is a cluster exercise: its check.yaml needs `shipped:`, what the first problem of the unsolved exercise says", ex.Name)
			}
			if !c.HasSteps() {
				t.Errorf("%s is a cluster exercise and needs steps in its check.yaml", ex.Name)
			}
			broken, _ := fs.ReadFile(k8slings.FS, ex.Source())
			sol, _ := fs.ReadFile(k8slings.Solutions, ex.SolutionPath())
			sources := [][]byte{broken, sol}
			if ex.Mode == exercises.ModeFix {
				sources = [][]byte{sol}
				files, err := exercises.SetupFiles(k8slings.FS, ex.Path)
				if err != nil {
					t.Fatal(err)
				}
				for _, f := range files {
					step, _ := fs.ReadFile(k8slings.FS, f)
					sources = append(sources, step)
				}
			}
			if shortcuts, _ := fs.Glob(k8slings.Solutions, path.Join(path.Dir(ex.SolutionPath()), "shortcut-*.sh")); len(shortcuts) > 0 {
				for _, f := range shortcuts {
					data, _ := fs.ReadFile(k8slings.Solutions, f)
					sources = append(sources, data)
				}
			}
			for _, src := range sources {
				for _, m := range imageRe.FindAllSubmatch(src, -1) {
					if !preloaded[string(m[1])] && !brokenOnPurpose[string(m[1])] {
						t.Errorf("%s uses the image %s, which cluster.PreloadImages does not list", ex.Name, m[1])
					}
				}
			}
		}
	}
}

// brokenOnPurpose are images a fix exercise may name although they are not
// preloaded: a typo nothing can pull (the lesson), and an image that only a
// shortcut tries, which the check refuses before anything is pulled.
var brokenOnPurpose = map[string]bool{"ngnix:1.27": true, "httpd:2.4": true, "nginx:1.27.99": true}

var rejectedRe = regexp.MustCompile(`^# rejected: "(.+)"$`)

// A fix exercise has no file to edit: a ticket (README.md), the broken state
// (setup.yaml) and the checks stay in the binary except the README, and the
// solution is a kubectl script. Each shortcut script says what the check must
// say when it is tried.
func testFixExercise(t *testing.T, ex exercises.Exercise) {
	t.Helper()
	for _, f := range []string{exercises.ReadmeFile, exercises.SetupFile, exercises.CheckFile} {
		if _, err := fs.Stat(k8slings.FS, path.Join(ex.Path, f)); err != nil {
			t.Errorf("missing %s: %v", f, err)
		}
	}
	if _, err := fs.Stat(k8slings.FS, path.Join(ex.Path, exercises.ExerciseFile)); err == nil {
		t.Errorf("a fix exercise has no %s: the learner repairs the cluster", exercises.ExerciseFile)
	}

	files, err := exercises.SetupFiles(k8slings.FS, ex.Path)
	if err != nil {
		t.Fatal(err)
	}
	validator := schema.New(k8slings.Schemas)
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
				var noSchema schema.ErrNoSchema
				if errors.As(err, &noSchema) {
					t.Errorf("no embedded schema for %s: add it to scripts/gen-schemas.sh", noSchema)
				}
			}
		}
	}

	raw, err := fs.ReadFile(k8slings.FS, ex.CheckPath())
	if err != nil {
		t.Fatal(err)
	}
	c, err := check.Load(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.UnchangedResources()) == 0 {
		t.Errorf("%s needs an `unchanged:` step: without it, deleting the broken object and creating a new one would pass", ex.Name)
	}

	script, err := fs.ReadFile(k8slings.Solutions, ex.SolutionPath())
	if err != nil {
		t.Fatalf("missing solution script: %v", err)
	}
	if strings.TrimSpace(string(script)) == "" {
		t.Error("the solution script is empty")
	}
	shortcuts, _ := fs.Glob(k8slings.Solutions, path.Join(path.Dir(ex.SolutionPath()), "shortcut-*.sh"))
	if len(shortcuts) == 0 {
		t.Errorf("%s needs at least one shortcut-*.sh: a way to cheat that the check must refuse", ex.Name)
	}
	for _, f := range shortcuts {
		data, err := fs.ReadFile(k8slings.Solutions, f)
		if err != nil {
			t.Fatal(err)
		}
		first, _, _ := strings.Cut(string(data), "\n")
		if !rejectedRe.MatchString(first) {
			t.Errorf(`%s must start with: # rejected: "what the check says"`, f)
		}
	}
}

// The setup steps of every fix exercise are setup.yaml, setup-2.yaml ... without
// a gap and within the limit, and are never given to the learner.
func TestSetupStepsAreConsecutive(t *testing.T) {
	for _, ex := range load(t) {
		if ex.Mode != exercises.ModeFix {
			if _, err := fs.Stat(k8slings.FS, ex.SetupPath()); err == nil {
				t.Errorf("%s is not a fix exercise and must not have a setup", ex.Name)
			}
			continue
		}
		if _, err := exercises.SetupFiles(k8slings.FS, ex.Path); err != nil {
			t.Errorf("%s: %v", ex.Name, err)
		}
	}
}
