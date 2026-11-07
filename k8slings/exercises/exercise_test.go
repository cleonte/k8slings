package exercises

import (
	"os"
	"path/filepath"
	"testing"
)

func writeExercise(t *testing.T, content string) Exercise {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	ex := Exercise{Name: "x", Path: "exercises/t/x", Topic: "t", Mode: ModeManifest, Hint: "h"}
	if err := os.MkdirAll(filepath.Dir(ex.File()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ex.File(), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return ex
}

func TestStateAndMarkDone(t *testing.T) {
	ex := writeExercise(t, "# I AM NOT DONE\nkind: Pod\n")
	if ex.State() != Pending {
		t.Fatal("an exercise with the marker is pending")
	}
	if err := ex.MarkDone(); err != nil {
		t.Fatal(err)
	}
	if ex.State() != Done {
		t.Fatal("the exercise should be done after the marker is removed")
	}
	if got, _ := os.ReadFile(ex.File()); string(got) != "kind: Pod\n" {
		t.Fatalf("only the marker line should go, got %q", got)
	}
	if err := ex.MarkDone(); err != ErrNotMarked {
		t.Fatalf("want ErrNotMarked, got %v", err)
	}
}

func TestParseInfoRejectsMistakes(t *testing.T) {
	for name, in := range map[string]string{
		"unknown mode": "[[exercises]]\nname='a'\npath='p'\ntopic='t'\nmode='magic'\nhint='h'\n",
		"duplicate":    "[[exercises]]\nname='a'\npath='p'\ntopic='t'\nmode='manifest'\nhint='h'\n[[exercises]]\nname='a'\npath='q'\ntopic='t'\nmode='manifest'\nhint='h'\n",
		"no hint":      "[[exercises]]\nname='a'\npath='p'\ntopic='t'\nmode='manifest'\n",
	} {
		if _, err := ParseInfo([]byte(in)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestParseObjectsMultiDocument(t *testing.T) {
	objs, err := ParseObjects([]byte("# only a comment\n---\nkind: A\n---\nkind: B\n---\n"))
	if err != nil || len(objs) != 2 {
		t.Fatalf("objs=%v err=%v", objs, err)
	}
}

func TestParseObjectsCRLF(t *testing.T) {
	objs, err := ParseObjects([]byte("# c\r\n---\r\nkind: A\r\n---\r\nkind: B\r\n"))
	if err != nil || len(objs) != 2 {
		t.Fatalf("objs=%v err=%v", objs, err)
	}
}

func writeFile(name, content string) error { return os.WriteFile(name, []byte(content), 0o644) }

func TestLocationPointsAtTheTodoSoThatATerminalCanLinkToIt(t *testing.T) {
	t.Chdir(t.TempDir())
	ex := Exercise{Name: "pods1", Path: "exercises/pods/pods1", Mode: ModeManifest}
	file := ex.File()
	if got := ex.Location(); got != file {
		t.Fatalf("without a file: %q, want %q", got, file)
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(s string) {
		if err := os.WriteFile(file, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("# I AM NOT DONE\nkind: Pod\nspec:\n  # TODO: fix the field\n  contianers: []\n")
	if got, want := ex.Location(), file+":4"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	write("# I AM NOT DONE\nkind: Pod\n")
	if got := ex.Location(); got != file {
		t.Fatalf("without a TODO: %q, want %q", got, file)
	}
	// A fix exercise has only its ticket, with no fix site in it.
	fix := Exercise{Name: "troubleshooting1", Path: "exercises/troubleshooting/troubleshooting1", Mode: ModeFix}
	if got := fix.Location(); got != fix.File() {
		t.Fatalf("fix: %q, want %q", got, fix.File())
	}
}
