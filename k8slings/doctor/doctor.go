// Package doctor diagnoses the learner's setup in two halves. Offline mode
// (the workspace and what the binary brings with it) needs nothing else, so
// its problems are failures. Cluster mode needs Docker, kind, kubectl and the
// cluster: when those are missing that is only a warning, because manifest
// exercises do not need them. A cluster that exists but is broken is a failure.
package doctor

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/cleonte/k8slings/k8slings/cluster"
	"github.com/cleonte/k8slings/k8slings/exercises"
)

type Status int

const (
	OK Status = iota
	Warn
	Fail
)

// The two halves of the report.
const (
	Offline     = "Offline mode"
	ClusterMode = "Cluster mode"
)

type Check struct {
	Group  string
	Name   string
	Status Status
	Detail string
}

// Env holds everything doctor learns about the machine, so tests can fake it.
type Env struct {
	FileExists func(path string) bool
	// Schemas is the embedded schema tree: schemas/VERSION and schemas/*.json.gz.
	Schemas fs.FS
	// Cluster is the k8slings cluster, for the cluster half. Nil skips it.
	Cluster *cluster.Cluster
}

func DefaultEnv(schemas fs.FS, cl *cluster.Cluster) Env {
	return Env{
		Cluster: cl,
		FileExists: func(path string) bool {
			_, err := os.Stat(path)
			return err == nil
		},
		Schemas: schemas,
	}
}

// Run executes all checks and returns their results in display order.
func Run(ctx context.Context, infoFile string, env Env) []Check {
	checks := offlineChecks(infoFile, env)
	if env.Cluster != nil {
		checks = append(checks, clusterChecks(ctx, env.Cluster)...)
	}
	if all, err := exercises.List(infoFile); err == nil {
		checks = append(checks, modesCheck(all, Ready(checks, ClusterMode) && env.Cluster != nil))
	}
	for i := range checks {
		if checks[i].Group == "" {
			checks[i].Group = Offline
		}
	}
	return checks
}

func offlineChecks(infoFile string, env Env) []Check {
	checks := []Check{schemaCheck(env)}

	all, err := exercises.List(infoFile)
	if err != nil {
		return append(checks, Check{
			Name:   "Exercise manifest",
			Status: Fail,
			Detail: fmt.Sprintf("can't read %s: %v (run k8slings from the folder that holds it, or create one with `k8slings init`)", infoFile, err),
		})
	}
	return append(checks,
		Check{Name: "Exercise manifest", Status: OK, Detail: fmt.Sprintf("%s lists %d exercises", infoFile, len(all))},
		filesCheck(all, env),
	)
}

// Ready reports whether a group has no failures and no warnings.
func Ready(checks []Check, group string) bool {
	for _, c := range checks {
		if c.Group == group && c.Status != OK {
			return false
		}
	}
	return true
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

// modesCheck says how many exercises need the cluster, and whether it is
// ready for them, so that "all done" is not mistaken for "everything there is".
func modesCheck(all []exercises.Exercise, clusterReady bool) Check {
	needs := 0
	for _, e := range all {
		if e.Mode != exercises.ModeManifest {
			needs++
		}
	}
	switch {
	case needs == 0:
		return Check{Group: Offline, Name: "Exercise modes", Status: OK, Detail: "every exercise runs offline, no cluster needed"}
	case clusterReady:
		return Check{Group: ClusterMode, Name: "Exercise modes", Status: OK, Detail: fmt.Sprintf("%d of %d exercises use the cluster, and it is ready", needs, len(all))}
	}
	// This is about cluster mode, not a problem of offline mode.
	return Check{
		Group:  ClusterMode,
		Name:   "Exercise modes",
		Status: Warn,
		Detail: fmt.Sprintf("%d of %d exercises need the cluster; until it is ready they are skipped", needs, len(all)),
	}
}
