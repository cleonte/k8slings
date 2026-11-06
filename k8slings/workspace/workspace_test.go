package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/cleonte/k8slings/k8slings/exercises"
)

var src = fstest.MapFS{
	"info.toml":                   {Data: []byte("[[exercises]]\nname='a'\npath='exercises/t/a'\ntopic='t'\nmode='manifest'\nhint='h'\n")},
	"exercises/t/a/exercise.yaml": {Data: []byte("# I AM NOT DONE\n")},
	"exercises/t/a/check.yaml":    {Data: []byte("objects: []\n")},
	"exercises/t/a/README.md":     {Data: []byte("# a\n")},
}

func TestInitKeepsCheckFilesOut(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "ws")
	if err := Init(src, dest); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"info.toml", "exercises/t/a/exercise.yaml", "exercises/t/a/README.md"} {
		if _, err := os.Stat(filepath.Join(dest, filepath.FromSlash(want))); err != nil {
			t.Errorf("missing %s: %v", want, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dest, "exercises", "t", "a", "check.yaml")); err == nil {
		t.Error("check.yaml must stay inside the binary")
	}
	if err := Init(src, dest); !errors.Is(err, ErrExists) {
		t.Errorf("a second init must refuse to overwrite, got %v", err)
	}
}

func TestResetRestoresMarker(t *testing.T) {
	root := t.TempDir()
	p, err := Reset(src, root, "a")
	if err != nil || p != "exercises/t/a/exercise.yaml" {
		t.Fatalf("p=%q err=%v", p, err)
	}
	got, _ := os.ReadFile(filepath.Join(root, "exercises", "t", "a", "exercise.yaml"))
	if string(got) != "# I AM NOT DONE\n" {
		t.Fatalf("got %q", got)
	}
}

var fixSrc = fstest.MapFS{
	"info.toml":                       {Data: []byte("[[exercises]]\nname='f'\npath='exercises/t/f'\ntopic='t'\nmode='fix'\nhint='h'\n")},
	"exercises/t/f/README.md":         {Data: []byte("# ticket\n")},
	"exercises/t/f/setup.yaml":        {Data: []byte("one\n")},
	"exercises/t/f/setup-2.yaml":      {Data: []byte("two\n")},
	"exercises/t/f/setup-3.yaml":      {Data: []byte("three\n")},
	"exercises/t/f/check.yaml":        {Data: []byte("steps: []\n")},
	"exercises/t/f/solution-note.txt": {Data: []byte("kept\n")},
}

func TestInitKeepsEverySetupStepOut(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "ws")
	if err := Init(fixSrc, dest); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"setup.yaml", "setup-2.yaml", "setup-3.yaml", "check.yaml"} {
		if _, err := os.Stat(filepath.Join(dest, "exercises", "t", "f", name)); err == nil {
			t.Errorf("%s must stay inside the binary", name)
		}
	}
	for _, name := range []string{"README.md", "solution-note.txt"} {
		if _, err := os.Stat(filepath.Join(dest, "exercises", "t", "f", name)); err != nil {
			t.Errorf("%s should have been written: %v", name, err)
		}
	}
}

// A fix exercise is reset in the cluster (cmd/reset.go), never from files. If a
// fix exercise ever reached Reset, it must not put the setup steps in the workspace.
func TestResetNeverWritesSetupFiles(t *testing.T) {
	root := t.TempDir()
	_, _ = Reset(fixSrc, root, "f")
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && exercises.EmbeddedOnly(d.Name()) {
			t.Errorf("reset wrote %s, which belongs in the binary", p)
		}
		return nil
	})
}
