// Package workspace creates a folder with the exercises and restores single
// exercises to their original state, from the copy embedded in the binary.
package workspace

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/cleonte/k8slings/k8slings/exercises"
)

var ErrExists = errors.New("already exists")

// Init writes the exercises in src to a new folder dest. It refuses to touch
// a path that already exists. The check.yaml files stay in the binary.
func Init(src fs.FS, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("%s %w", dest, ErrExists)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := write(src, dest); err != nil {
		_ = os.RemoveAll(dest)
		return err
	}
	return nil
}

func write(src fs.FS, dest string) error {
	return fs.WalkDir(src, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Names inside an fs.FS always use forward slashes.
		target := filepath.Join(dest, filepath.FromSlash(name))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if d.Name() == exercises.CheckFile {
			return nil
		}
		data, err := fs.ReadFile(src, name)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// Reset restores the exercise file of the exercise called name below root to
// the version in src, which also brings back its marker. It returns the path
// of the restored file, relative to root and with forward slashes.
func Reset(src fs.FS, root, name string) (string, error) {
	info, err := fs.ReadFile(src, "info.toml")
	if err != nil {
		return "", err
	}
	all, err := exercises.ParseInfo(info)
	if err != nil {
		return "", err
	}
	for _, ex := range all {
		if ex.Name != name {
			continue
		}
		// The path comes from the manifest, never from the user's input.
		if !fs.ValidPath(ex.Path) {
			return "", fmt.Errorf("exercise %s has an invalid path %q", ex.Name, ex.Path)
		}
		original, err := fs.ReadFile(src, ex.Source())
		if err != nil {
			return "", err
		}
		target := filepath.Join(root, filepath.FromSlash(ex.Source()))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		return path.Clean(ex.Source()), os.WriteFile(target, original, 0o644)
	}
	return "", exercises.ErrExerciseNotFound
}
