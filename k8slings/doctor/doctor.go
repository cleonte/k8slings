// Package doctor diagnoses the learner's setup: the exercises workspace and
// what the binary brings with it. Offline mode needs nothing else; the checks
// for Docker, kind and the cluster arrive with cluster mode.
package doctor

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/cleonte/k8slings/k8slings/exercises"
)

type Status int

const (
	OK Status = iota
	Warn
	Fail
)

type Check struct {
	Name   string
	Status Status
	Detail string
}

// Env holds everything doctor learns about the machine, so tests can fake it.
type Env struct {
	FileExists func(path string) bool
	// Schemas is the embedded schema tree: schemas/VERSION and schemas/*.json.gz.
	Schemas fs.FS
}

func DefaultEnv(schemas fs.FS) Env {
	return Env{
		FileExists: func(path string) bool {
			_, err := os.Stat(path)
			return err == nil
		},
		Schemas: schemas,
	}
}

// Run executes all checks and returns their results in display order.
func Run(infoFile string, env Env) []Check {
	checks := []Check{schemaCheck(env)}

	all, err := exercises.List(infoFile)
	if err != nil {
		return append(checks, Check{
			Name:   "Exercise manifest",
			Status: Fail,
			Detail: fmt.Sprintf("can't read %s: %v (run k8slings from the folder that holds it, or create one with `k8slings init`)", infoFile, err),
		})
	}
	checks = append(checks,
		Check{Name: "Exercise manifest", Status: OK, Detail: fmt.Sprintf("%s lists %d exercises", infoFile, len(all))},
		filesCheck(all, env),
		modesCheck(all),
	)
	return checks
}

// Failed reports whether any check failed (warnings don't count).
func Failed(checks []Check) bool {
	for _, c := range checks {
		if c.Status == Fail {
			return true
		}
	}
	return false
}

func schemaCheck(env Env) Check {
	version, err := fs.ReadFile(env.Schemas, "schemas/VERSION")
	if err != nil {
		return Check{Name: "Kubernetes schemas", Status: Fail, Detail: "the binary has no embedded schemas: " + err.Error()}
	}
	files, _ := fs.Glob(env.Schemas, "schemas/*.json.gz")
	return Check{
		Name:   "Kubernetes schemas",
		Status: OK,
		Detail: fmt.Sprintf("Kubernetes %s, %d kinds, built into the binary (no network needed)", strings.TrimSpace(string(version)), len(files)),
	}
}

// filesCheck makes sure every exercise file the manifest lists is there.
func filesCheck(all []exercises.Exercise, env Env) Check {
	var missing []string
	for _, e := range all {
		if !env.FileExists(e.File()) {
			missing = append(missing, e.File())
		}
	}
	if len(missing) == 0 {
		return Check{Name: "Exercise files", Status: OK, Detail: "every exercise file is in place"}
	}
	return Check{
		Name:   "Exercise files",
		Status: Fail,
		Detail: fmt.Sprintf("%d missing, for example %s (restore one with `k8slings reset <name>`)", len(missing), missing[0]),
	}
}

// modesCheck says how many exercises need the cluster, so that "all done" is
// not mistaken for "everything there is".
func modesCheck(all []exercises.Exercise) Check {
	cluster := 0
	for _, e := range all {
		if e.Mode != exercises.ModeManifest {
			cluster++
		}
	}
	if cluster == 0 {
		return Check{Name: "Exercise modes", Status: OK, Detail: "every exercise runs offline, no cluster needed"}
	}
	return Check{
		Name:   "Exercise modes",
		Status: Warn,
		Detail: fmt.Sprintf("%d of %d exercises need the cluster (`k8slings cluster up`); until it exists they are skipped", cluster, len(all)),
	}
}
