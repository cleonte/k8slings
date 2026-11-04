package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
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
