package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/cleonte/k8slings/k8slings/workspace"
)

func tree(names ...string) fstest.MapFS {
	info := ""
	fsys := fstest.MapFS{}
	for _, n := range names {
		info += "[[exercises]]\nname = \"" + n + "\"\npath = \"exercises/t/" + n + "\"\ntopic = \"t\"\nmode = \"manifest\"\nhint = \"h\"\n\n"
		fsys["exercises/t/"+n+"/exercise.yaml"] = &fstest.MapFile{Data: []byte("# I AM NOT DONE\n")}
		fsys["exercises/t/"+n+"/README.md"] = &fstest.MapFile{Data: []byte("# " + n + "\n")}
	}
	fsys["info.toml"] = &fstest.MapFile{Data: []byte(info)}
	return fsys
}

func TestTheNoticeAppearsOnlyWhenTheFolderIsBehind(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ws")
	if err := workspace.Init(tree("a", "b"), root); err != nil {
		t.Fatal(err)
	}
	if got := newExercisesLine(tree("a", "b"), root); got != "" {
		t.Fatalf("an up to date folder has no notice: %q", got)
	}
	got := newExercisesLine(tree("a", "b", "c", "d", "e", "f"), root)
	for _, want := range []string{"4 exercises", "c, d, e and 1 more", "k8slings update"} {
		if !strings.Contains(got, want) {
			t.Errorf("the notice %q should say %q", got, want)
		}
	}
	if got := newExercisesLine(tree("a"), filepath.Join(t.TempDir(), "nowhere")); got != "" {
		t.Errorf("a folder that is not a workspace has no notice: %q", got)
	}
}

func TestUpdateCommandAddsAndRefusesOutsideAWorkspace(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ws")
	if err := workspace.Init(tree("a"), root); err != nil {
		t.Fatal(err)
	}
	c := UpdateCmd(tree("a", "b"), root)
	if err := c.RunE(c, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "exercises", "t", "b", "exercise.yaml")); err != nil {
		t.Fatalf("update did not add b: %v", err)
	}
	c = UpdateCmd(tree("a", "b"), filepath.Join(t.TempDir(), "nowhere"))
	if err := c.RunE(c, nil); err == nil {
		t.Fatal("update outside a workspace must fail")
	}
}
