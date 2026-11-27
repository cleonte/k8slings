package workspace

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/cleonte/k8slings/k8slings/exercises"
	lw "github.com/cleonte/lingscore/workspace"
)

// UpdateResult says what Update did (or, for a dry run, would do).
type UpdateResult struct {
	// Added are the exercises of the binary that the workspace did not have.
	Added []exercises.Exercise
	// Readmes is how many README.md files were replaced by a newer text.
	Readmes int
	// Refreshed are exercises whose file was replaced by a newer version because
	// the learner had not touched the old one.
	Refreshed []string
	// Newer are exercises whose file has a newer version that was NOT applied,
	// because the learner has changed the old one: `reset` brings it, and loses
	// their work.
	Newer []string
}

func readInfo(src fs.FS, root string) (embedded []exercises.Exercise, embeddedText string, workspace []exercises.Exercise, workspaceText string, err error) {
	raw, err := fs.ReadFile(src, "info.toml")
	if err != nil {
		return nil, "", nil, "", err
	}
	embedded, err = exercises.ParseInfo(raw)
	if err != nil {
		return nil, "", nil, "", err
	}
	mine, err := os.ReadFile(filepath.Join(root, "info.toml"))
	if err != nil {
		return nil, "", nil, "", err
	}
	workspace, err = exercises.ParseInfo(mine)
	if err != nil {
		return nil, "", nil, "", err
	}
	return embedded, string(raw), workspace, string(mine), nil
}

// Missing returns the names of the exercises that the binary brings and the
// workspace in root does not list yet, in the order of the binary. It never
// fails loudly: a workspace that cannot be read has nothing to compare.
func Missing(src fs.FS, root string) []string {
	embedded, _, mine, _, err := readInfo(src, root)
	if err != nil {
		return nil
	}
	have := map[string]bool{}
	for _, e := range mine {
		have[e.Name] = true
	}
	var missing []string
	for _, e := range embedded {
		if !have[e.Name] {
			missing = append(missing, e.Name)
		}
	}
	return missing
}

// Update brings the workspace in root up to date with the exercises in src: it
// adds the exercises the workspace does not have, in the order of the binary,
// and refreshes what the learner cannot have changed. It never overwrites a file
// of an existing exercise that may hold the learner's work, and it keeps a copy
// of info.toml as info.toml.bak. With dryRun it only reports.
func Update(src fs.FS, root string, dryRun bool) (UpdateResult, error) {
	var res UpdateResult
	embedded, embeddedText, mine, mineText, err := readInfo(src, root)
	if err != nil {
		return res, err
	}
	have := map[string]bool{}
	for _, e := range mine {
		have[e.Name] = true
	}
	originals := exercises.Originals(root)
	record := map[string]string{}

	// Exercises that are already there: newer READMEs (nobody edits them), and
	// newer exercise files where that is safe.
	for _, e := range embedded {
		if !have[e.Name] {
			continue
		}
		if data, err := fs.ReadFile(src, path.Join(e.Path, exercises.ReadmeFile)); err == nil {
			target := filepath.Join(root, filepath.FromSlash(e.Path), exercises.ReadmeFile)
			if old, err := os.ReadFile(target); err != nil || string(old) != string(data) {
				res.Readmes++
				if !dryRun {
					if err := lw.WriteFile(target, data, true); err != nil {
						return res, err
					}
				}
			}
		}
		if e.Mode == exercises.ModeFix {
			continue
		}
		original, err := fs.ReadFile(src, e.Source())
		if err != nil {
			continue
		}
		target := filepath.Join(root, filepath.FromSlash(e.Source()))
		current, err := os.ReadFile(target)
		if err != nil {
			continue
		}
		recorded, known := originals[e.Name]
		switch lw.Decide(current, original, recorded, known) {
		case lw.Record:
			record[e.Name] = lw.Hash(original) // untouched and current: a later release can refresh it
		case lw.Refresh:
			res.Refreshed = append(res.Refreshed, e.Name) // untouched, and the binary has a newer one
			if !dryRun {
				if err := lw.WriteFile(target, original, true); err != nil {
					return res, err
				}
				record[e.Name] = lw.Hash(original)
			}
		case lw.Newer:
			res.Newer = append(res.Newer, e.Name) // the learner has edited it
		}
	}

	// Exercises that are not there yet.
	for _, e := range embedded {
		if !have[e.Name] {
			res.Added = append(res.Added, e)
		}
	}
	var added []string
	for _, e := range res.Added {
		added = append(added, e.Name)
		if dryRun {
			continue
		}
		if err := copyExercise(src, root, e, record); err != nil {
			return res, err
		}
	}

	if len(res.Added) > 0 && !dryRun {
		merged := lw.MergeInfo(embeddedText, mineText, exerciseNames(embedded), exerciseNames(res.Added), nil)
		if err := os.WriteFile(filepath.Join(root, "info.toml.bak"), []byte(mineText), 0o644); err != nil {
			return res, err
		}
		if err := os.WriteFile(filepath.Join(root, "info.toml"), []byte(merged), 0o644); err != nil {
			return res, err
		}
	}
	if !dryRun && (len(record) > 0 || len(added) > 0) {
		if err := exercises.RecordOriginals(root, record, added); err != nil {
			return res, err
		}
	}
	return res, nil
}

// copyExercise writes the files of one exercise into the workspace, except the
// ones that stay in the binary, and never over a file that is already there.
func copyExercise(src fs.FS, root string, e exercises.Exercise, record map[string]string) error {
	return fs.WalkDir(src, e.Path, func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || exercises.EmbeddedOnly(name) {
			return err
		}
		data, err := fs.ReadFile(src, name)
		if err != nil {
			return err
		}
		if err := lw.WriteFile(filepath.Join(root, filepath.FromSlash(name)), data, false); err != nil {
			return err
		}
		if e.Mode != exercises.ModeFix && name == e.Source() {
			record[e.Name] = lw.Hash(data)
		}
		return nil
	})
}

// exerciseNames are the names of the exercises, in order.
func exerciseNames(list []exercises.Exercise) []string {
	out := make([]string, len(list))
	for i, e := range list {
		out[i] = e.Name
	}
	return out
}

// Describe is a short phrase for a list of exercises: "config5, config6 and 24 more".
func Describe(names []string) string { return lw.Describe(names) }
