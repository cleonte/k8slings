package cluster

import "testing"

func TestEditorCommandSplitsTheEditorAndAddsTheFile(t *testing.T) {
	if EditorCommand("", "a.yaml") != nil || EditorCommand("  ", "a.yaml") != nil {
		t.Fatal("no editor, no command")
	}
	c := EditorCommand("code -w", "exercises/pods/pods1/exercise.yaml")
	want := []string{"code", "-w", "exercises/pods/pods1/exercise.yaml"}
	if len(c.Args) != len(want) {
		t.Fatalf("args = %v, want %v", c.Args, want)
	}
	for i := range want {
		if c.Args[i] != want[i] {
			t.Fatalf("args = %v, want %v", c.Args, want)
		}
	}
}
