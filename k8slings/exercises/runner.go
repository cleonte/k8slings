package exercises

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"

	"github.com/cleonte/k8slings/k8slings/check"
	"github.com/cleonte/k8slings/k8slings/schema"
	"sigs.k8s.io/yaml"
)

// Env is what running an exercise needs: the embedded exercise tree (for the
// check.yaml files, which are never written to the workspace) and the schemas.
type Env struct {
	Embedded  fs.FS // the k8slings.FS: info.toml and exercises/
	Validator *schema.Validator
}

// Result is what a successful run reports.
type Result struct {
	Exercise Exercise
	Objects  int
}

// FailedError means the exercise is not solved. Lines are written for the
// learner, one problem per line.
type FailedError struct{ Lines []string }

func (e *FailedError) Error() string { return strings.Join(e.Lines, "\n") }

// SkippedError means the exercise cannot run here, and says why.
type SkippedError struct{ Reason string }

func (e *SkippedError) Error() string { return e.Reason }

func IsSkipped(err error) bool {
	var s *SkippedError
	return errors.As(err, &s)
}

// Run checks the learner's exercise file. It does not look at the
// "I AM NOT DONE" marker; callers decide what a pending-but-passing exercise
// means.
func (e Exercise) Run(env *Env) (Result, error) {
	src, err := os.ReadFile(e.File())
	if err != nil {
		return Result{}, fmt.Errorf("cannot read %s: %w", e.File(), err)
	}
	return e.RunSource(env, src)
}

// RunSource checks src as if it were the content of the exercise file.
func (e Exercise) RunSource(env *Env, src []byte) (Result, error) {
	if e.Mode != ModeManifest {
		return Result{}, &SkippedError{fmt.Sprintf("%s is a %s exercise; that mode is not available yet", e.Name, e.Mode)}
	}
	raw, err := fs.ReadFile(env.Embedded, e.CheckPath())
	if err != nil {
		return Result{}, fmt.Errorf("%s has no check.yaml: %w", e.Name, err)
	}
	c, err := check.Load(raw)
	if err != nil {
		return Result{}, fmt.Errorf("%s: %w", e.Name, err)
	}
	objs, err := ParseObjects(src)
	if err != nil {
		return Result{}, &FailedError{[]string{"Your YAML could not be read: " + err.Error()}}
	}
	if len(objs) == 0 {
		return Result{}, &FailedError{[]string{"The file has no Kubernetes objects."}}
	}

	if c.Schema == "strict" {
		var problems []string
		for _, o := range objs {
			ps, err := env.Validator.Validate(o)
			if err != nil {
				return Result{}, fmt.Errorf("%s: %w", e.Name, err)
			}
			for _, p := range ps {
				problems = append(problems, fmt.Sprintf("%s %s: %s", o["kind"], objectName(o), p))
			}
		}
		if len(problems) > 0 {
			return Result{}, &FailedError{problems}
		}
	}
	if failures := c.Run(objs); len(failures) > 0 {
		return Result{}, &FailedError{failures}
	}
	return Result{Exercise: e, Objects: len(objs)}, nil
}

// Windows editors and git checkouts write CRLF, so the separator line may end in \r.
var docSeparator = regexp.MustCompile(`(?m)^---[ \t]*\r?$`)

// ParseObjects splits a (possibly multi-document) YAML file into objects.
// Documents with nothing but comments are skipped.
func ParseObjects(src []byte) ([]map[string]any, error) {
	var objs []map[string]any
	for i, doc := range docSeparator.Split(string(src), -1) {
		var m map[string]any
		if err := yaml.Unmarshal([]byte(doc), &m); err != nil {
			return nil, fmt.Errorf("document %d: %w", i+1, err)
		}
		if m != nil {
			objs = append(objs, m)
		}
	}
	return objs, nil
}

func objectName(o map[string]any) string {
	md, _ := o["metadata"].(map[string]any)
	n, _ := md["name"].(string)
	if n == "" {
		return "(unnamed)"
	}
	return n
}
