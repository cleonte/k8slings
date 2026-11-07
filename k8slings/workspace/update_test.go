package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/cleonte/k8slings/k8slings/exercises"
)

// entry is the info.toml text of one exercise.
func entry(name, mode string) string {
	return "[[exercises]]\nname = \"" + name + "\"\npath = \"exercises/t/" + name + "\"\ntopic = \"t\"\nmode = \"" + mode + "\"\nhint = \"h\"\n\n"
}

// release builds the embedded tree of a version of k8slings with these exercises
// (in order); exercise files say which version of the file they are.
func release(version string, names ...string) fstest.MapFS {
	info := "# the course\n\n"
	fsys := fstest.MapFS{}
	for _, n := range names {
		mode := "manifest"
		if strings.HasPrefix(n, "fix") {
			mode = "fix"
		}
		info += entry(n, mode)
		fsys["exercises/t/"+n+"/README.md"] = &fstest.MapFile{Data: []byte("# " + n + " " + version + "\n")}
		fsys["exercises/t/"+n+"/check.yaml"] = &fstest.MapFile{Data: []byte("steps: []\n")}
		if mode == "fix" {
			fsys["exercises/t/"+n+"/setup.yaml"] = &fstest.MapFile{Data: []byte("kind: Pod\n")}
		} else {
			fsys["exercises/t/"+n+"/exercise.yaml"] = &fstest.MapFile{Data: []byte("# I AM NOT DONE\n# " + n + " " + version + "\n")}
		}
	}
	fsys["info.toml"] = &fstest.MapFile{Data: []byte(info)}
	return fsys
}

func names(t *testing.T, root string) []string {
	t.Helper()
	all, err := exercises.List(filepath.Join(root, "info.toml"))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range all {
		out = append(out, e.Name)
	}
	return out
}

func read(t *testing.T, parts ...string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(parts...))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func workspaceOf(t *testing.T, fsys fstest.MapFS) string {
	t.Helper()
	dest := filepath.Join(t.TempDir(), "ws")
	if err := Init(fsys, dest); err != nil {
		t.Fatal(err)
	}
	return dest
}

func TestUpdateAddsNewExercisesWhereTheCourseHasThem(t *testing.T) {
	root := workspaceOf(t, release("1", "a", "b", "c"))
	next := release("1", "x", "a", "b", "n1", "n2", "c", "fix1", "z")

	res, err := Update(next, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(names(t, root), " "), "x a b n1 n2 c fix1 z"; got != want {
		t.Fatalf("order = %s, want %s", got, want)
	}
	if len(res.Added) != 5 {
		t.Fatalf("added %d, want 5", len(res.Added))
	}
	// The files come with their marker; what stays in the binary is not written.
	if got := read(t, root, "exercises", "t", "n1", "exercise.yaml"); !strings.HasPrefix(got, "# I AM NOT DONE") {
		t.Errorf("a new exercise has no marker: %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "exercises", "t", "fix1", "README.md")); err != nil {
		t.Errorf("the ticket of a fix exercise is missing: %v", err)
	}
	for _, hidden := range []string{"exercises/t/n1/check.yaml", "exercises/t/fix1/setup.yaml"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(hidden))); err == nil {
			t.Errorf("%s must stay inside the binary", hidden)
		}
	}
	// The header and the earlier entries are untouched, and the copy is kept.
	if !strings.HasPrefix(read(t, root, "info.toml"), "# the course\n\n") {
		t.Error("what comes before the first exercise was changed")
	}
	if got := read(t, root, "info.toml.bak"); !strings.Contains(got, "name = \"c\"") || strings.Contains(got, "n1") {
		t.Errorf("info.toml.bak is not the old file:\n%s", got)
	}
	if !exercises.IsNewIn(root, "n1") || exercises.IsNewIn(root, "a") {
		t.Error("the added exercises are marked new, and only they")
	}
}

func TestUpdateNeverOverwritesWorkAndIsIdempotent(t *testing.T) {
	root := workspaceOf(t, release("1", "a", "b"))
	work := filepath.Join(root, "exercises", "t", "a", "exercise.yaml")
	if err := os.WriteFile(work, []byte("my own work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	next := release("2", "a", "b", "c")
	if _, err := Update(next, root, false); err != nil {
		t.Fatal(err)
	}
	if got := read(t, work); got != "my own work\n" {
		t.Fatalf("the learner's work was changed: %q", got)
	}
	before := read(t, root, "info.toml")
	res, err := Update(next, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Added) != 0 || read(t, root, "info.toml") != before {
		t.Fatalf("a second update must change nothing: %+v", res)
	}
}

func TestUpdateRefreshesReadmesAndFilesTheLearnerHasNotTouched(t *testing.T) {
	root := workspaceOf(t, release("1", "a", "b", "c"))
	// b has been edited, a and c have not.
	b := filepath.Join(root, "exercises", "t", "b", "exercise.yaml")
	if err := os.WriteFile(b, []byte("my work on b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	next := release("2", "a", "b", "c")
	res, err := Update(next, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Readmes != 3 || read(t, root, "exercises", "t", "a", "README.md") != "# a 2\n" {
		t.Errorf("the READMEs were not refreshed: %+v", res)
	}
	if got := read(t, root, "exercises", "t", "a", "exercise.yaml"); !strings.Contains(got, "a 2") {
		t.Errorf("an untouched file should be replaced by the newer one: %q", got)
	}
	if got := read(t, b); got != "my work on b\n" {
		t.Errorf("an edited file must stay: %q", got)
	}
	if strings.Join(res.Newer, ",") != "b" || strings.Join(res.Refreshed, ",") != "a,c" {
		t.Errorf("refreshed %v, newer %v", res.Refreshed, res.Newer)
	}
	// Refreshed files are the originals again: the next release can refresh them too.
	res, err = Update(release("3", "a", "b", "c"), root, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(res.Refreshed, ",") != "a,c" {
		t.Errorf("the second release refreshed %v", res.Refreshed)
	}
}

func TestUpdateOfAnOldWorkspaceWithoutHashesOnlyAdds(t *testing.T) {
	root := workspaceOf(t, release("1", "a"))
	if err := os.Remove(filepath.Join(root, exercises.ProgressFile)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "exercises", "t", "a", "exercise.yaml"), []byte("old version or my work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Update(release("2", "a", "b"), root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Added) != 1 || len(res.Refreshed) != 0 || len(res.Newer) != 0 {
		t.Fatalf("without a record the file cannot be judged: %+v", res)
	}
	if got := read(t, root, "exercises", "t", "a", "exercise.yaml"); got != "old version or my work\n" {
		t.Errorf("the file was touched: %q", got)
	}
}

func TestUpdateDryRunChangesNothing(t *testing.T) {
	root := workspaceOf(t, release("1", "a"))
	before := read(t, root, "info.toml")
	res, err := Update(release("2", "a", "b"), root, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Added) != 1 || res.Readmes != 1 {
		t.Fatalf("a dry run still says what it would do: %+v", res)
	}
	if read(t, root, "info.toml") != before || read(t, root, "exercises", "t", "a", "README.md") != "# a 1\n" {
		t.Error("a dry run changed the folder")
	}
	if _, err := os.Stat(filepath.Join(root, "exercises", "t", "b")); err == nil {
		t.Error("a dry run wrote files")
	}
	if _, err := os.Stat(filepath.Join(root, "info.toml.bak")); err == nil {
		t.Error("a dry run wrote a backup")
	}
}

func TestUpdateLeavesExercisesThatThisVersionDoesNotKnow(t *testing.T) {
	root := workspaceOf(t, release("1", "a", "old"))
	if _, err := Update(release("2", "a", "new"), root, false); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(names(t, root), " "); got != "a new old" && got != "a old new" {
		t.Fatalf("names = %s", got)
	}
	if _, err := os.Stat(filepath.Join(root, "exercises", "t", "old", "exercise.yaml")); err != nil {
		t.Errorf("an exercise this version no longer has was removed: %v", err)
	}
}

func TestMissingNamesWhatTheWorkspaceLacks(t *testing.T) {
	root := workspaceOf(t, release("1", "a", "b"))
	if got := Missing(release("1", "a", "b"), root); len(got) != 0 {
		t.Errorf("nothing is missing: %v", got)
	}
	if got := strings.Join(Missing(release("2", "a", "b", "c", "d"), root), ","); got != "c,d" {
		t.Errorf("missing = %s", got)
	}
	if got := Missing(release("2", "a"), filepath.Join(t.TempDir(), "nowhere")); got != nil {
		t.Errorf("a folder that is not a workspace has nothing to compare: %v", got)
	}
}

func TestResetRecordsTheOriginalAgain(t *testing.T) {
	fsys := release("1", "a")
	root := workspaceOf(t, fsys)
	file := filepath.Join(root, "exercises", "t", "a", "exercise.yaml")
	if err := os.WriteFile(file, []byte("my work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Reset(fsys, root, "a"); err != nil {
		t.Fatal(err)
	}
	res, err := Update(release("2", "a"), root, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(res.Refreshed, ",") != "a" {
		t.Errorf("after a reset the file is the original and can be refreshed: %+v", res)
	}
}
