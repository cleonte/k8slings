package exercises

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

// ProgressFile is where progress is kept that cannot live in an exercise file.
// A manifest or cluster exercise is done when its `I AM NOT DONE` line is gone;
// a fix exercise has no file to edit, so its completion is recorded here, next
// to info.toml. It also holds the identity (UID) of the objects a fix exercise
// seeded, so that "the same Deployment" can be told from one that was deleted
// and created again.
const ProgressFile = ".k8slings.json"

const progressFormat = 2

type progressData struct {
	// Format is 2 once passes are recorded for every mode (before, a manifest or
	// cluster exercise was done as soon as its marker was gone, passing or not).
	Format int `json:"format,omitempty"`
	// Done maps the name of an exercise to the time its check passed. A fix
	// exercise is done when it is here; a manifest or cluster exercise when it is
	// here and its marker is gone. Once done, an exercise stays done, even if the
	// learner breaks it again later.
	Done map[string]time.Time `json:"done,omitempty"`
	// Seeds maps a fix exercise to the UIDs of the objects it seeded, by "kind/name".
	Seeds map[string]map[string]string `json:"seeds,omitempty"`
	// Originals maps an exercise to the SHA-256 of its file as k8slings wrote it
	// (init, update). It is how `update` tells a file the learner has not touched
	// (safe to replace with a newer version) from one with their work in it.
	Originals map[string]string `json:"originals,omitempty"`
	// New lists the exercises that `update` added, so that `list` can mark them.
	New []string `json:"new,omitempty"`
}

// The file is read and written by several commands and by the full-screen view,
// so access in one process is serialised, and a write replaces the file whole.
var progressMu sync.Mutex

func readProgress() progressData { return readProgressIn(".") }

func readProgressIn(dir string) progressData {
	var p progressData
	data, err := os.ReadFile(filepath.Join(dir, ProgressFile))
	if err != nil {
		return p
	}
	// A damaged file means no progress, not a crash: the exercises can be redone.
	_ = json.Unmarshal(data, &p)
	return p
}

func writeProgress(p progressData) error { return writeProgressIn(".", p) }

func writeProgressIn(dir string, p progressData) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".k8slings-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(dir, ProgressFile))
}

func updateProgress(f func(*progressData)) error {
	progressMu.Lock()
	defer progressMu.Unlock()
	p := readProgress()
	f(&p)
	p.Format = progressFormat
	return writeProgress(p)
}

func passed(name string) bool {
	progressMu.Lock()
	defer progressMu.Unlock()
	_, ok := readProgress().Done[name]
	return ok
}

// markPassed records that the check of an exercise passed.
func markPassed(name string) error {
	return updateProgress(func(p *progressData) {
		if p.Done == nil {
			p.Done = map[string]time.Time{}
		}
		if _, already := p.Done[name]; !already {
			p.Done[name] = time.Now().UTC().Truncate(time.Second)
		}
	})
}

// recordSeed stores the UIDs of the objects a fix exercise has just seeded, and
// forgets that the exercise was done (a reset starts it over).
func recordSeed(name string, uids map[string]string) error {
	return updateProgress(func(p *progressData) {
		if p.Seeds == nil {
			p.Seeds = map[string]map[string]string{}
		}
		p.Seeds[name] = uids
		delete(p.Done, name)
	})
}

// forgetSeed drops the recorded UIDs before an exercise is seeded again: if the
// seeding then fails or is cancelled, the exercise must not look seeded.
func forgetSeed(name string) error {
	return updateProgress(func(p *progressData) { delete(p.Seeds, name) })
}

// hasSeed reports whether this workspace seeded the exercise (and so knows which
// objects were the original ones).
func hasSeed(name string) bool {
	progressMu.Lock()
	defer progressMu.Unlock()
	_, ok := readProgress().Seeds[name]
	return ok
}

// seedUID returns the UID recorded when the exercise was seeded.
func seedUID(name, resource string) (string, bool) {
	progressMu.Lock()
	defer progressMu.Unlock()
	uid, ok := readProgress().Seeds[name][resource]
	return uid, ok
}

// Originals returns the recorded hashes of the exercise files written into the
// workspace in dir.
func Originals(dir string) map[string]string {
	progressMu.Lock()
	defer progressMu.Unlock()
	return readProgressIn(dir).Originals
}

// RecordOriginals remembers the hashes of exercise files that k8slings has just
// written into the workspace in dir, and which exercises `update` has just added.
func RecordOriginals(dir string, hashes map[string]string, added []string) error {
	progressMu.Lock()
	defer progressMu.Unlock()
	p := readProgressIn(dir)
	if p.Originals == nil {
		p.Originals = map[string]string{}
	}
	for name, h := range hashes {
		p.Originals[name] = h
	}
	for _, name := range added {
		if !slices.Contains(p.New, name) {
			p.New = append(p.New, name)
		}
	}
	return writeProgressIn(dir, p)
}

// IsNew reports whether `update` added the exercise to this workspace.
func IsNew(name string) bool { return IsNewIn(".", name) }

// IsNewIn is IsNew for the workspace in dir.
func IsNewIn(dir, name string) bool {
	progressMu.Lock()
	defer progressMu.Unlock()
	return slices.Contains(readProgressIn(dir).New, name)
}

// StampFormat marks the workspace in dir as one that records passes from the
// start, so that no exercise in it is mistaken for one finished the old way.
func StampFormat(dir string) error {
	progressMu.Lock()
	defer progressMu.Unlock()
	p := readProgressIn(dir)
	p.Format = progressFormat
	return writeProgressIn(dir, p)
}

// ForgetPassed drops the recorded pass of an exercise in the workspace in dir
// (its file was reset, so it has to pass again).
func ForgetPassed(dir, name string) error {
	progressMu.Lock()
	defer progressMu.Unlock()
	p := readProgressIn(dir)
	if _, ok := p.Done[name]; !ok {
		return nil
	}
	delete(p.Done, name)
	return writeProgressIn(dir, p)
}

// grandfather credits the exercises whose marker was already gone before passes
// were recorded: they were done by the old rule, and a learner who upgrades must
// not find them pending again. It runs once per workspace.
func grandfather(all []Exercise) {
	progressMu.Lock()
	defer progressMu.Unlock()
	p := readProgress()
	if p.Format >= progressFormat {
		return
	}
	now := time.Now().UTC().Truncate(time.Second)
	changed := false
	for _, e := range all {
		if e.Mode == ModeFix {
			continue
		}
		if data, err := os.ReadFile(e.File()); err == nil && !notDoneRegex.Match(data) {
			if p.Done == nil {
				p.Done = map[string]time.Time{}
			}
			if _, ok := p.Done[e.Name]; !ok {
				p.Done[e.Name] = now
				changed = true
			}
		}
	}
	if _, err := os.Stat(ProgressFile); err != nil && !changed {
		return // not a workspace (yet): leave no file behind
	}
	p.Format = progressFormat
	_ = writeProgress(p)
}

// RecordPass records that the check of the exercise passed, as running it does.
func RecordPass(name string) error { return markPassed(name) }
