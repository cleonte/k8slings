package cluster

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// The safety rule of k8slings is that it only talks to its own cluster. That
// can only be enforced if there is one place that runs programs. This test
// fails if any other non-test file imports os/exec.
func TestOnlyExecGoRunsPrograms(t *testing.T) {
	root := filepath.Join("..", "..") // the repository root
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == "prototype" || name == ".git" || name == "bin" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			if imp.Path.Value == `"os/exec"` && filepath.ToSlash(path) != "../../k8slings/cluster/exec.go" {
				t.Errorf("%s imports os/exec: run programs through cluster.Runner instead", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
