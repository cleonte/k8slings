package exercises

import (
	"os"
	"time"

	"github.com/cleonte/lingscore/marker"
	"github.com/cleonte/lingscore/progress"
)

// ProgressFile is where progress is kept that cannot live in an exercise file.
// A manifest or cluster exercise is done when its `I AM NOT DONE` line is gone;
// a fix exercise has no file to edit, so its completion is recorded here, next
// to info.toml. It also holds the identity (UID) of the objects a fix exercise
// seeded, so that "the same Deployment" can be told from one that was deleted
// and created again.
const ProgressFile = ".k8slings.json"

const progressFormat = 2

// The file is kept by lingscore's progress package: the done passes, the hashes
// of the exercise files and the exercises `update` added are its own keys. What
// is k8slings' own lives under two more keys, which the package keeps through
// every write:
//
//   - "format" is 2 once passes are recorded for every mode (before, a manifest
//     or cluster exercise was done as soon as its marker was gone, passing or
//     not).
//   - "seeds" maps a fix exercise to the UIDs of the objects it seeded, by
//     "kind/name", so that "the same Deployment" can be told from one that was
//     deleted and created again.
//
// A manifest or cluster exercise is done when its pass is recorded and its
// `I AM NOT DONE` line is gone; a fix exercise has no file to edit, so its
// completion is the recorded pass.
const (
	keyFormat = "format"
	keySeeds  = "seeds"
)

// progressData is what the file holds.
type progressData = progress.Data

func store(dir string) progress.Store { return progress.Store{Dir: dir, File: ProgressFile} }

func readProgress() progressData { return readProgressIn(".") }

func readProgressIn(dir string) progressData { return store(dir).Read() }

// format is the format of the file: 0 for one that does not say.
func format(d progressData) int {
	var f int
	d.Get(keyFormat, &f)
	return f
}

func updateProgress(f func(*progressData)) error {
	return store(".").Update(func(p *progressData) {
		f(p)
		_ = p.Set(keyFormat, progressFormat)
	})
}

func passed(name string) bool { return store(".").Passed(name) }

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

func seeds(p progressData) map[string]map[string]string {
	var m map[string]map[string]string
	p.Get(keySeeds, &m)
	return m
}

// recordSeed stores the UIDs of the objects a fix exercise has just seeded, and
// forgets that the exercise was done (a reset starts it over).
func recordSeed(name string, uids map[string]string) error {
	return updateProgress(func(p *progressData) {
		m := seeds(*p)
		if m == nil {
			m = map[string]map[string]string{}
		}
		m[name] = uids
		_ = p.Set(keySeeds, m)
		delete(p.Done, name)
	})
}

// forgetSeed drops the recorded UIDs before an exercise is seeded again: if the
// seeding then fails or is cancelled, the exercise must not look seeded.
func forgetSeed(name string) error {
	return updateProgress(func(p *progressData) {
		m := seeds(*p)
		delete(m, name)
		_ = p.Set(keySeeds, m)
	})
}

// hasSeed reports whether this workspace seeded the exercise (and so knows which
// objects were the original ones).
func hasSeed(name string) bool {
	_, ok := seeds(readProgress())[name]
	return ok
}

// seedUID returns the UID recorded when the exercise was seeded.
func seedUID(name, resource string) (string, bool) {
	uid, ok := seeds(readProgress())[name][resource]
	return uid, ok
}

// Originals returns the recorded hashes of the exercise files written into the
// workspace in dir.
func Originals(dir string) map[string]string { return store(dir).Originals() }

// RecordOriginals remembers the hashes of exercise files that k8slings has just
// written into the workspace in dir, and which exercises `update` has just added.
func RecordOriginals(dir string, hashes map[string]string, added []string) error {
	return store(dir).RecordOriginals(hashes, added)
}

// IsNew reports whether `update` added the exercise to this workspace.
func IsNew(name string) bool { return IsNewIn(".", name) }

// IsNewIn is IsNew for the workspace in dir.
func IsNewIn(dir, name string) bool { return store(dir).IsNew(name) }

// StampFormat marks the workspace in dir as one that records passes from the
// start. `init` calls it.
func StampFormat(dir string) error {
	return store(dir).Update(func(p *progressData) { _ = p.Set(keyFormat, progressFormat) })
}

// ForgetPassed drops the recorded pass of an exercise in the workspace in dir
// (its file was reset, so it has to pass again).
func ForgetPassed(dir, name string) error { return store(dir).ForgetPass(name) }

// grandfather credits the exercises whose marker was already gone before passes
// were recorded: they were done by the old rule, and a learner who upgrades must
// not find them pending again. It runs once per workspace.
func grandfather(all []Exercise) {
	s := store(".")
	if format(s.Read()) >= progressFormat {
		return
	}
	var credit []string
	for _, e := range all {
		if e.Mode != ModeFix && marker.Gone(e.File(), notDoneRegex) {
			credit = append(credit, e.Name)
		}
	}
	_, err := os.Stat(ProgressFile)
	if err != nil && len(credit) == 0 {
		return // not a workspace (yet): leave no file behind
	}
	now := time.Now().UTC().Truncate(time.Second)
	_ = s.Update(func(p *progressData) {
		if format(*p) >= progressFormat {
			return
		}
		for _, name := range credit {
			if p.Done == nil {
				p.Done = map[string]time.Time{}
			}
			if _, ok := p.Done[name]; !ok {
				p.Done[name] = now
			}
		}
		_ = p.Set(keyFormat, progressFormat)
	})
}

// RecordPass records that the check of the exercise passed, as running it does.
func RecordPass(name string) error { return markPassed(name) }
