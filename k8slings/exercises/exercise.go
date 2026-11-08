// Package exercises loads the manifest (info.toml), tracks which exercises
// are done and runs them.
package exercises

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cleonte/k8slings/k8slings/cluster"
	"github.com/pelletier/go-toml/v2"
)

// Modes an exercise can run in: manifest needs no cluster, cluster applies the
// exercise to one, fix breaks the cluster for the learner to repair, query has
// the learner write one kubectl command and compares what it prints.
const (
	ModeManifest = "manifest"
	ModeCluster  = "cluster"
	ModeFix      = "fix"
	ModeQuery    = "query"
)

// File names inside an exercise folder.
const (
	ExerciseFile = "exercise.yaml"
	CheckFile    = "check.yaml"
	// SetupFile is the broken state of a fix exercise. Like check.yaml it stays
	// in the binary: it would give away what is wrong.
	SetupFile  = "setup.yaml"
	ReadmeFile = "README.md"
	// ScriptFile is the solution of a fix exercise: a kubectl script.
	ScriptFile = "solution.sh"
	// AnswerFile is what the learner writes in a query exercise, and the
	// reference answer in solutions/. It is not a script and is never run by a
	// shell (k8slings reads it), which is why it is not called .sh.
	AnswerFile = "answer.kubectl"
	// VariantFile adds objects to the state of a query exercise, to prove an
	// answer does not depend on the names and values of the first one.
	VariantFile = "setup-variant.yaml"
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
	// Offline marks a query exercise that needs no cluster (kubectl create ...
	// --dry-run=client). It has no setup.yaml.
	Offline bool
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
		case e.Mode != ModeManifest && e.Mode != ModeCluster && e.Mode != ModeFix && e.Mode != ModeQuery:
			return nil, fmt.Errorf("info.toml: exercise %q has unknown mode %q", e.Name, e.Mode)
		case e.Offline && e.Mode != ModeQuery:
			return nil, fmt.Errorf("info.toml: exercise %q is offline, which only a query exercise can be", e.Name)
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
	all, err := ParseInfo(data)
	if err == nil {
		grandfather(all)
	}
	return all, err
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
func NextPending(infoFile string, av Availability) (Exercise, error) {
	all, err := List(infoFile)
	if err != nil {
		return Exercise{}, err
	}
	for _, e := range all {
		if e.State() == Pending && e.SkipReason(av) == "" {
			return e, nil
		}
	}
	return Exercise{}, ErrNoPending
}

var todoLineRe = regexp.MustCompile(`#\s*TODO:`)

// Location is File with the line of the first `# TODO:` comment added
// ("exercises/pods/pods1/exercise.yaml:7"), which the terminal of an editor such
// as VS Code turns into a link that opens the file at the fix site. Without a
// file, or a TODO in it, it is File.
func (e Exercise) Location() string {
	file := e.File()
	if e.Mode == ModeFix {
		return file
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return file
	}
	for i, line := range strings.Split(string(data), "\n") {
		if todoLineRe.MatchString(line) {
			return fmt.Sprintf("%s:%d", file, i+1)
		}
	}
	return file
}

// Namespace is the namespace a cluster exercise runs in: one per exercise, so
// that cleaning up is trivial and exercises cannot disturb each other.
func (e Exercise) Namespace() string { return cluster.NamespacePrefix + e.Name }

// File is the path, on disk, of the file the learner edits. A fix exercise has
// no such file: the learner works on the cluster, and the file that matters is
// the README.md with the ticket.
func (e Exercise) File() string {
	if e.Mode == ModeFix {
		return filepath.FromSlash(path.Join(e.Path, ReadmeFile))
	}
	return filepath.FromSlash(e.Source())
}

// EmbeddedOnly says whether a file of an exercise folder stays in the binary and
// is never written to the learner's workspace: the checks, and the setup steps
// (setup.yaml, setup-2.yaml, ...), because they would give the cause away. Init,
// reset and doctor all decide with this one function.
func EmbeddedOnly(name string) bool {
	base := path.Base(name)
	return base == CheckFile || base == SetupFile || base == VariantFile || setupStepRe.MatchString(base)
}

var setupStepRe = regexp.MustCompile(`^setup-([0-9]+)\.yaml$`)

// MaxSetupSteps is the most steps a fix exercise's setup can have.
const MaxSetupSteps = 5

// SetupFiles lists the setup steps of the exercise folder dir in fsys, in order:
// setup.yaml, then setup-2.yaml, setup-3.yaml and so on without a gap. It
// reports a gap, a file beyond MaxSetupSteps, or a folder without setup.yaml.
func SetupFiles(fsys fs.FS, dir string) ([]string, error) {
	files := []string{path.Join(dir, SetupFile)}
	if _, err := fs.Stat(fsys, files[0]); err != nil {
		return nil, fmt.Errorf("%s has no %s", dir, SetupFile)
	}
	for n := 2; ; n++ {
		f := path.Join(dir, fmt.Sprintf("setup-%d.yaml", n))
		if _, err := fs.Stat(fsys, f); err != nil {
			break
		}
		files = append(files, f)
	}
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	extra := 0
	for _, e := range entries {
		if setupStepRe.MatchString(e.Name()) {
			extra++
		}
	}
	if extra != len(files)-1 {
		return nil, fmt.Errorf("%s: the setup steps must be setup.yaml, setup-2.yaml, setup-3.yaml ... without a gap", dir)
	}
	if len(files) > MaxSetupSteps {
		return nil, fmt.Errorf("%s has %d setup steps, the most allowed is %d", dir, len(files), MaxSetupSteps)
	}
	return files, nil
}

// SetupPath is the broken state of a fix exercise, in the embedded tree.
func (e Exercise) SetupPath() string { return path.Join(e.Path, SetupFile) }

// Embedded names inside the embedded exercise tree (always forward slashes).
func (e Exercise) CheckPath() string { return path.Join(e.Path, CheckFile) }
func (e Exercise) Source() string {
	if e.Mode == ModeQuery {
		return path.Join(e.Path, AnswerFile)
	}
	return path.Join(e.Path, ExerciseFile)
}

// SolutionPath is where the solution lives in the embedded solutions tree.
func (e Exercise) SolutionPath() string {
	if e.Mode == ModeFix {
		return path.Join("solutions", strings.TrimPrefix(e.Path, "exercises/"), ScriptFile)
	}
	if e.Mode == ModeQuery {
		return path.Join("solutions", strings.TrimPrefix(e.Path, "exercises/"), AnswerFile)
	}
	return path.Join("solutions", strings.TrimPrefix(e.Path, "exercises/"), ExerciseFile)
}

type State int

const (
	Pending State = iota + 1
	Done
)

func (s State) String() string { return [...]string{"Pending", "Done"}[s-1] }

// State is Done when the check has passed (a fix exercise), or has passed and
// the learner has removed the marker (the others). Taking the marker out of a
// file that does not pass yet does not skip the exercise.
func (e Exercise) State() State {
	if !passed(e.Name) {
		return Pending
	}
	if e.Mode == ModeFix {
		return Done
	}
	if e.MarkerGone() {
		return Done
	}
	return Pending
}

// MarkerGone reports whether the exercise file exists without the marker line.
func (e Exercise) MarkerGone() bool {
	if e.Mode == ModeFix {
		return false
	}
	data, err := os.ReadFile(e.File())
	return err == nil && !notDoneRegex.Match(data)
}

// MarkDone removes the marker line, which is what a learner does by hand.
func (e Exercise) MarkDone() error {
	if e.Mode == ModeFix {
		return nil // a fix exercise is done when its check passes, which records it
	}
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
