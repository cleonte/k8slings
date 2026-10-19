package main

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:exercises
var exerciseFS embed.FS

const wsFile = ".k8slings.json"

// Workspace is the directory the learner edits exercises in.
type Workspace struct {
	Root     string          `json:"-"`
	Progress map[string]bool `json:"progress"`
}

func findWorkspace() (*Workspace, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for {
		p := filepath.Join(dir, wsFile)
		if b, err := os.ReadFile(p); err == nil {
			ws := &Workspace{Root: dir, Progress: map[string]bool{}}
			if err := json.Unmarshal(b, ws); err != nil {
				return nil, err
			}
			if ws.Progress == nil {
				ws.Progress = map[string]bool{}
			}
			return ws, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, errors.New("not inside a k8slings workspace; run `k8slings init` first")
		}
		dir = parent
	}
}

func (w *Workspace) save() error {
	b, _ := json.MarshalIndent(w, "", "  ")
	return os.WriteFile(filepath.Join(w.Root, wsFile), b, 0o644)
}

func (w *Workspace) exercisePath(ex *Exercise) string {
	return filepath.Join(w.Root, "exercises", ex.ID, "exercise.yaml")
}

func (w *Workspace) doneCount() int {
	n := 0
	for _, ex := range registry {
		if w.Progress[ex.ID] {
			n++
		}
	}
	return n
}

func (w *Workspace) next() *Exercise {
	for _, ex := range registry {
		if !w.Progress[ex.ID] {
			return ex
		}
	}
	return nil
}

// extract copies embedded exercises into the workspace. With overwrite=false
// existing files are left alone (so re-running init never destroys work).
func (w *Workspace) extract(only string, overwrite bool) error {
	return fs.WalkDir(exerciseFS, "exercises", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if only != "" && path != "exercises" && !hasPrefixDir(path, "exercises/"+only) {
			return nil
		}
		dst := filepath.Join(w.Root, path)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		if _, err := os.Stat(dst); err == nil && !overwrite {
			return nil
		}
		b, err := exerciseFS.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, 0o644)
	})
}

func hasPrefixDir(path, dir string) bool {
	return path == dir || len(path) > len(dir) && path[:len(dir)+1] == dir+"/"
}
