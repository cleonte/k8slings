package exercises

import (
	"encoding/json"
	"os"
	"path/filepath"
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

type progressData struct {
	// Done maps the name of a fix exercise to the time it was fixed. Once done,
	// an exercise stays done, even if the learner breaks it again later.
	Done map[string]time.Time `json:"done,omitempty"`
	// Seeds maps a fix exercise to the UIDs of the objects it seeded, by "kind/name".
	Seeds map[string]map[string]string `json:"seeds,omitempty"`
}

// The file is read and written by several commands and by the full-screen view,
// so access in one process is serialised, and a write replaces the file whole.
var progressMu sync.Mutex

func readProgress() progressData {
	var p progressData
	data, err := os.ReadFile(ProgressFile)
	if err != nil {
		return p
	}
	// A damaged file means no progress, not a crash: the exercises can be redone.
	_ = json.Unmarshal(data, &p)
	return p
}

func writeProgress(p progressData) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(ProgressFile), ".k8slings-*.tmp")
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
	return os.Rename(tmp.Name(), ProgressFile)
}

func updateProgress(f func(*progressData)) error {
	progressMu.Lock()
	defer progressMu.Unlock()
	p := readProgress()
	f(&p)
	return writeProgress(p)
}

func fixDone(name string) bool {
	progressMu.Lock()
	defer progressMu.Unlock()
	_, ok := readProgress().Done[name]
	return ok
}

// markFixDone records that a fix exercise is fixed.
func markFixDone(name string) error {
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
