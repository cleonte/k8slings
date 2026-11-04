// Package exercises loads the manifest (info.toml), tracks which exercises
// are done and runs them.
package exercises

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Modes an exercise can run in. Only manifest is implemented so far.
const (
	ModeManifest = "manifest"
	ModeCluster  = "cluster"
	ModeFix      = "fix"
)

// File names inside an exercise folder.
const (
	ExerciseFile = "exercise.yaml"
	CheckFile    = "check.yaml"
)

var notDoneRegex = regexp.MustCompile(`(?m)^\s*#\s*I\s+AM\s+NOT\s+DONE`)

// Exercise is one entry of info.toml.
type Exercise struct {
	Name  string
	Path  string // folder, with forward slashes, relative to the workspace
	Topic string
	Mode  string
	Hint  string
	Docs  string
	// K8sVersion is the minimum Kubernetes minor version (e.g. "1.37") needed.
	K8sVersion string `toml:"k8s_version"`
	// Requires lists add-ons or topology the exercise needs (cluster modes).
	Requires []string
	// Timeout is how long cluster and fix modes wait for the check to pass.
	Timeout string
}

var (
	ErrExerciseNotFound = errors.New("exercise not found")
	ErrNotMarked        = errors.New("the exercise has no 'I AM NOT DONE' marker")
	ErrNoPending        = errors.New("no pending exercises: you are done")
)

// ParseInfo parses and validates the content of an info.toml.
func ParseInfo(data []byte) ([]Exercise, error) {
	var info struct{ Exercises []Exercise }
	if err := toml.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("info.toml: %w", err)
	}
	seen := map[string]bool{}
	for _, e := range info.Exercises {
		switch {
		case e.Name == "":
			return nil, errors.New("info.toml: an exercise has no name")
		case seen[e.Name]:
			return nil, fmt.Errorf("info.toml: duplicate exercise %q", e.Name)
		case e.Path == "" || e.Topic == "" || e.Hint == "":
			return nil, fmt.Errorf("info.toml: exercise %q needs path, topic and hint", e.Name)
		case e.Mode != ModeManifest && e.Mode != ModeCluster && e.Mode != ModeFix:
			return nil, fmt.Errorf("info.toml: exercise %q has unknown mode %q", e.Name, e.Mode)
		}
		seen[e.Name] = true
	}
	return info.Exercises, nil
}

// List reads the manifest file.
func List(infoFile string) ([]Exercise, error) {
	data, err := os.ReadFile(infoFile)
	if err != nil {
		return nil, err
	}
	return ParseInfo(data)
}

// Find returns the exercise called name.
func Find(name, infoFile string) (Exercise, error) {
	all, err := List(infoFile)
	if err != nil {
		return Exercise{}, err
	}
	for _, e := range all {
		if e.Name == name {
			return e, nil
		}
	}
	return Exercise{}, ErrExerciseNotFound
}

// NextPending returns the first exercise, in manifest order, that is not done
// and can run here.
func NextPending(infoFile string) (Exercise, error) {
	all, err := List(infoFile)
	if err != nil {
		return Exercise{}, err
	}
	for _, e := range all {
		if e.State() == Pending && e.SkipReason() == "" {
			return e, nil
		}
	}
	return Exercise{}, ErrNoPending
}

// File is the path, on disk, of the file the learner edits.
func (e Exercise) File() string { return filepath.FromSlash(path.Join(e.Path, ExerciseFile)) }

// Embedded names inside the embedded exercise tree (always forward slashes).
func (e Exercise) CheckPath() string { return path.Join(e.Path, CheckFile) }
func (e Exercise) Source() string    { return path.Join(e.Path, ExerciseFile) }

// SolutionPath is where the solution lives in the embedded solutions tree.
func (e Exercise) SolutionPath() string {
	return path.Join("solutions", strings.TrimPrefix(e.Path, "exercises/"), ExerciseFile)
}

type State int

const (
	Pending State = iota + 1
	Done
)

func (s State) String() string { return [...]string{"Pending", "Done"}[s-1] }

// State is Pending while the exercise file has the marker (or is missing).
func (e Exercise) State() State {
	data, err := os.ReadFile(e.File())
	if err != nil || notDoneRegex.Match(data) {
		return Pending
	}
	return Done
}

// MarkDone removes the marker line, which is what a learner does by hand.
func (e Exercise) MarkDone() error {
	info, err := os.Stat(e.File())
	if err != nil {
		return err
	}
	data, err := os.ReadFile(e.File())
	if err != nil {
		return err
	}
	lines := strings.SplitAfter(string(data), "\n")
	kept := make([]string, 0, len(lines))
	removed := false
	for _, line := range lines {
		if notDoneRegex.MatchString(line) {
			removed = true
			continue
		}
		kept = append(kept, line)
	}
	if !removed {
		return ErrNotMarked
	}
	return os.WriteFile(e.File(), []byte(strings.Join(kept, "")), info.Mode().Perm())
}
