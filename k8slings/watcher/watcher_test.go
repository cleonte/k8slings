package watcher_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cleonte/k8slings/k8slings/watcher"
)

// quiet is long enough that a slow machine does not split quick writes in two.
const quiet = 250 * time.Millisecond

func start(t *testing.T) (dir string, changes <-chan []string, stop context.CancelFunc) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "exercises")
	dir = filepath.Join(root, "a")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	changes, err := watcher.Watch(ctx, root, quiet)
	if err != nil {
		t.Fatal(err)
	}
	return dir, changes, cancel
}

func write(t *testing.T, name, content string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func next(t *testing.T, changes <-chan []string) []string {
	t.Helper()
	select {
	case batch, ok := <-changes:
		if !ok {
			t.Fatal("the channel was closed")
		}
		return batch
	case <-time.After(5 * time.Second):
		t.Fatal("no change reported")
		return nil
	}
}

func expectQuiet(t *testing.T, changes <-chan []string) {
	t.Helper()
	select {
	case batch := <-changes:
		t.Fatalf("unexpected change: %v", batch)
	case <-time.After(3 * quiet):
	}
}

func TestSaveIsReportedOnce(t *testing.T) {
	dir, changes, _ := start(t)
	file := filepath.Join(dir, "exercise.yaml")
	for range 5 {
		write(t, file, "kind: Pod\n")
	}
	if batch := next(t, changes); len(batch) != 1 || batch[0] != file {
		t.Fatalf("got %v, want [%s]", batch, file)
	}
	expectQuiet(t, changes)
}

// A query exercise is answered in answer.kubectl: saving it must re-run the check.
func TestSavingAnAnswerOfAQueryExerciseIsReported(t *testing.T) {
	dir, changes, _ := start(t)
	file := filepath.Join(dir, "answer.kubectl")
	write(t, file, "kubectl get pods\n")
	if batch := next(t, changes); len(batch) != 1 || batch[0] != file {
		t.Fatalf("got %v, want [%s]", batch, file)
	}
	write(t, filepath.Join(dir, ".answer.kubectl.swp"), "x\n")
	write(t, filepath.Join(dir, ".answer.kubectl"), "x\n")
	expectQuiet(t, changes)
}

func TestAtomicSaveByRename(t *testing.T) {
	dir, changes, _ := start(t)
	file := filepath.Join(dir, "exercise.yaml")
	write(t, file, "a\n")
	next(t, changes)

	tmp := filepath.Join(dir, ".exercise.yaml.swp.yaml")
	write(t, tmp, "b\n")
	if err := os.Rename(tmp, file); err != nil {
		t.Fatal(err)
	}
	if batch := next(t, changes); len(batch) != 1 || batch[0] != file {
		t.Fatalf("got %v, want [%s]", batch, file)
	}
}

func TestOtherFilesAreIgnored(t *testing.T) {
	dir, changes, _ := start(t)
	write(t, filepath.Join(dir, "README.md"), "x\n")
	write(t, filepath.Join(dir, ".hidden.yaml"), "x\n")
	write(t, filepath.Join(dir, "#backup.yaml"), "x\n")
	expectQuiet(t, changes)
}

func TestNewFoldersAreWatched(t *testing.T) {
	dir, changes, _ := start(t)
	sub := filepath.Join(filepath.Dir(dir), "b")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	time.Sleep(quiet) // let the watcher add the folder
	file := filepath.Join(sub, "exercise.yaml")
	write(t, file, "x\n")
	if batch := next(t, changes); len(batch) != 1 || batch[0] != file {
		t.Fatalf("got %v, want [%s]", batch, file)
	}
}

func TestStopClosesTheChannel(t *testing.T) {
	_, changes, stop := start(t)
	stop()
	select {
	case _, ok := <-changes:
		if ok {
			t.Fatal("expected the channel to be closed")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the channel was not closed")
	}
}
