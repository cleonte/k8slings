package exercises

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/cleonte/k8slings/k8slings/check"
	"github.com/cleonte/k8slings/k8slings/cluster"
	"github.com/cleonte/k8slings/k8slings/schema"
	"sigs.k8s.io/yaml"
)

// Kube is the k8slings cluster, as far as running an exercise needs it. It is
// implemented by cluster.Cluster; tests use a fake.
type Kube interface {
	check.Kube
	// Verify says whether the kubeconfig leads to the cluster's own API server.
	Verify(ctx context.Context) error
	ResetNamespace(ctx context.Context, ns string) error
	Apply(ctx context.Context, ns string, manifest []byte) (string, error)
	// Delete removes the objects of a manifest, and waits until they are gone.
	Delete(ctx context.Context, ns string, manifest []byte) error
	// Query runs a read-only kubectl command that the query package has checked.
	Query(ctx context.Context, args []string, offline bool) (cluster.QueryResult, error)
	Nodes(ctx context.Context) (int, error)
	// ReadyAddons says, for each add-on of the registry, whether it is installed
	// and ready.
	ReadyAddons(ctx context.Context) map[string]bool
	Diagnose(ctx context.Context, ns string) []string
	// For fix exercises: bring a namespace to its broken state, say whether it
	// is in it, and tell the identity of an object.
	Seed(ctx context.Context, ns, name string, steps [][]byte) error
	Seeded(ctx context.Context, ns, name string) bool
	UID(ctx context.Context, ns, resource string) (string, error)
}

// Env is what running an exercise needs: the embedded exercise tree (for the
// check.yaml files, which are never written to the workspace), the schemas and,
// for cluster exercises, the cluster.
type Env struct {
	Embedded fs.FS // the k8slings.FS: info.toml and exercises/
	// Solutions is the embedded solutions/ tree. A query exercise needs it when it
	// runs: the reference answer is what the learner's command is compared with.
	Solutions fs.FS
	Validator *schema.Validator
	// Kube is nil when there is no cluster support; cluster exercises are then skipped.
	Kube Kube
	// Poll is how often a cluster check is repeated; default 2 seconds.
	Poll time.Duration
	// Timeout, when set, replaces the timeout of every cluster exercise. The
	// proof that an unsolved exercise fails uses a short one.
	Timeout time.Duration
	// Settle is how long a repaired fix exercise must keep passing before it
	// counts; default 3 seconds, and a negative value means no wait. A crashing
	// container is briefly Running and Ready before it exits, so one look that
	// happens to land there would call a broken cluster fixed.
	Settle time.Duration
}

func (e *Env) settle() time.Duration {
	switch {
	case e.Settle < 0:
		return 0
	case e.Settle == 0:
		return 3 * time.Second
	}
	return e.Settle
}

// DefaultTimeout is how long a cluster exercise has to start working.
const DefaultTimeout = 60 * time.Second

// Availability looks at what this machine can run right now.
func (e *Env) Availability(ctx context.Context) Availability {
	if e == nil || e.Kube == nil || e.Kube.Verify(ctx) != nil {
		return Availability{}
	}
	n, _ := e.Kube.Nodes(ctx)
	return Availability{Cluster: true, Nodes: n, Addons: e.Kube.ReadyAddons(ctx)}
}

// Result is what a successful run reports.
type Result struct {
	Exercise Exercise
	Objects  int
}

// FailedError means the exercise is not solved. Lines are written for the
// learner, one problem per line.
type FailedError struct {
	Lines []string
	// Details is what the cluster looks like, for a learner who is stuck: not
	// problems themselves, so it is shown without bullets.
	Details []string
}

func (e *FailedError) Error() string { return strings.Join(e.Lines, "\n") }

// SeededError is not a failure: it says that the cluster has just been brought
// to the broken state of a fix exercise, and that it is now the learner's turn.
type SeededError struct {
	Exercise  Exercise
	Namespace string
}

func (e *SeededError) Error() string {
	return fmt.Sprintf("%s: the cluster is now broken on purpose, in the namespace %s", e.Exercise.Name, e.Namespace)
}

// IsSeeded reports whether err says that an exercise was just seeded.
func IsSeeded(err error) bool {
	var s *SeededError
	return errors.As(err, &s)
}

type noSeedKey struct{}

// NoSeed marks a context: running a fix exercise with it only looks at the
// cluster, and never brings it to the broken state. Checking everything
// (`verify`, check-all) must not break the cluster for exercises the learner
// has not started.
func NoSeed(ctx context.Context) context.Context {
	return context.WithValue(ctx, noSeedKey{}, true)
}

func noSeed(ctx context.Context) bool { v, _ := ctx.Value(noSeedKey{}).(bool); return v }

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
func (e Exercise) Run(ctx context.Context, env *Env) (Result, error) {
	if e.Mode == ModeFix {
		return e.RunSource(ctx, env, nil) // there is no file to read
	}
	src, err := os.ReadFile(e.File())
	if err != nil {
		return Result{}, fmt.Errorf("cannot read %s: %w", e.File(), err)
	}
	res, err := e.RunSource(ctx, env, src)
	if err == nil {
		if err := markPassed(e.Name); err != nil {
			return Result{}, fmt.Errorf("it passes, but cannot save your progress in %s: %w", ProgressFile, err)
		}
	}
	return res, err
}

// RunSource checks src as if it were the content of the exercise file.
func (e Exercise) RunSource(ctx context.Context, env *Env, src []byte) (Result, error) {
	if e.Mode == ModeCluster || e.Mode == ModeFix || e.Mode == ModeQuery {
		if reason := e.SkipReason(env.Availability(ctx)); reason != "" {
			return Result{}, &SkippedError{fmt.Sprintf("%s %s", e.Name, reason)}
		}
	}
	raw, err := fs.ReadFile(env.Embedded, e.CheckPath())
	if err != nil {
		return Result{}, fmt.Errorf("%s has no check.yaml: %w", e.Name, err)
	}
	c, err := check.Load(raw)
	if err != nil {
		return Result{}, fmt.Errorf("%s: %w", e.Name, err)
	}
	if e.Mode == ModeFix {
		return e.runFix(ctx, env, c)
	}
	if e.Mode == ModeQuery {
		return e.runQuery(ctx, env, c, src)
	}
	objs, err := ParseObjects(src)
	if err != nil {
		return Result{}, &FailedError{Lines: []string{"Your YAML could not be read: " + err.Error()}}
	}
	if len(objs) == 0 {
		return Result{}, &FailedError{Lines: []string{"The file has no Kubernetes objects."}}
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
			return Result{}, &FailedError{Lines: problems}
		}
	}
	if failures := c.Run(objs); len(failures) > 0 {
		return Result{}, &FailedError{Lines: failures}
	}
	if e.Mode == ModeCluster {
		if err := e.runCluster(ctx, env, c, src); err != nil {
			return Result{}, err
		}
	}
	return Result{Exercise: e, Objects: len(objs)}, nil
}

// runFix looks at the cluster and says whether the learner has repaired it. The
// first time, the cluster is brought to its broken state, which is reported as
// a SeededError and not as a failure. After that nothing is ever seeded again by
// itself: that would wipe the repair. The check is a single look, not a wait:
// the learner is working in the cluster and the full-screen view looks again
// every couple of seconds.
func (e Exercise) runFix(ctx context.Context, env *Env, c *check.Compiled) (Result, error) {
	ns := e.Namespace()
	// The cluster counts as seeded only if this workspace seeded it. A namespace
	// left over from another folder, or from before the progress file was lost,
	// cannot be judged (nobody knows which objects were the original ones), so the
	// exercise starts fresh instead of leaving the learner stuck.
	if !env.Kube.Seeded(ctx, ns, e.Name) || !hasSeed(e.Name) {
		if noSeed(ctx) {
			return Result{}, &SkippedError{fmt.Sprintf("%s has not been started: run it to break the cluster on purpose", e.Name)}
		}
		if err := e.seed(ctx, env, c); err != nil {
			return Result{}, err
		}
		return Result{}, &SeededError{Exercise: e, Namespace: ns}
	}

	problem := c.RunSteps(ctx, env.Kube, ns, fixSeeds{e.Name})
	if problem.Message != "" {
		failed := &FailedError{Lines: []string{problem.Message}}
		if ctx.Err() == nil {
			failed.Details = env.Kube.Diagnose(ctx, ns)
		}
		return Result{}, failed
	}

	// A repair must hold. Look again after a moment, and only then call it fixed.
	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	case <-time.After(env.settle()):
	}
	if again := c.RunSteps(ctx, env.Kube, ns, fixSeeds{e.Name}); again.Message != "" {
		failed := &FailedError{Lines: []string{again.Message + " (it looked fine a moment ago, but it did not hold)"}}
		if ctx.Err() == nil {
			failed.Details = env.Kube.Diagnose(ctx, ns)
		}
		return Result{}, failed
	}
	if err := markPassed(e.Name); err != nil {
		return Result{}, fmt.Errorf("fixed, but cannot save your progress in %s: %w", ProgressFile, err)
	}
	return Result{Exercise: e}, nil
}

// Reseed puts a fix exercise back into its broken state, and forgets that it
// was fixed. It is what `reset` does for exercises that have no file.
func (e Exercise) Reseed(ctx context.Context, env *Env) error {
	if e.Mode != ModeFix && !(e.Mode == ModeQuery && !e.Offline) {
		return fmt.Errorf("%s has no cluster state to put back", e.Name)
	}
	raw, err := fs.ReadFile(env.Embedded, e.CheckPath())
	if err != nil {
		return err
	}
	c, err := check.Load(raw)
	if err != nil {
		return err
	}
	if env.Kube == nil || env.Kube.Verify(ctx) != nil {
		return &SkippedError{fmt.Sprintf("%s needs the cluster: run `k8slings cluster up`", e.Name)}
	}
	if e.Mode == ModeQuery {
		if !e.hasSetup(env) {
			return nil // nothing to put back
		}
		return e.seedQuery(ctx, env, c)
	}
	return e.seed(ctx, env, c)
}

func (e Exercise) seed(ctx context.Context, env *Env, c *check.Compiled) error {
	files, err := SetupFiles(env.Embedded, e.Path)
	if err != nil {
		return err
	}
	var steps [][]byte
	for _, f := range files {
		step, err := fs.ReadFile(env.Embedded, f)
		if err != nil {
			return fmt.Errorf("%s has no %s: %w", e.Name, path.Base(f), err)
		}
		steps = append(steps, step)
	}
	ns := e.Namespace()
	// Until the seeding has finished, nothing may look seeded: a failed or
	// cancelled run must be started over, not mistaken for the earlier one.
	if err := forgetSeed(e.Name); err != nil {
		return err
	}
	if err := env.Kube.Seed(ctx, ns, e.Name, steps); err != nil {
		return err
	}
	// Remember which objects were there, so that deleting one and creating it
	// again cannot pass for a repair.
	uids := map[string]string{}
	for _, res := range c.UnchangedResources() {
		uid, err := env.Kube.UID(ctx, ns, res)
		if err != nil {
			return fmt.Errorf("%s was not created by the setup: %w", res, err)
		}
		uids[res] = uid
	}
	return recordSeed(e.Name, uids)
}

// fixSeeds reads the UIDs recorded when an exercise was seeded.
type fixSeeds struct{ name string }

func (f fixSeeds) UID(resource string) (string, bool) { return seedUID(f.name, resource) }

// timeout is how long the cluster has to make the exercise work.
func (e Exercise) timeout(env *Env) time.Duration {
	if env.Timeout > 0 {
		return env.Timeout
	}
	if d, err := time.ParseDuration(e.Timeout); err == nil && d > 0 {
		return d
	}
	return DefaultTimeout
}

// runCluster applies the manifest to the exercise's own namespace and polls the
// steps of the check until they all hold or the time is up. The static checks
// (schema, assertions on the manifest) have already passed, so a typo is
// reported in milliseconds, without touching the cluster.
func (e Exercise) runCluster(ctx context.Context, env *Env, c *check.Compiled, src []byte) error {
	ns := e.Namespace()
	if err := env.Kube.ResetNamespace(ctx, ns); err != nil {
		return err
	}
	if out, err := env.Kube.Apply(ctx, ns, src); err != nil {
		// Some mistakes are rejected by the API server when the manifest is applied.
		return &FailedError{Lines: applyProblems(out)}
	}

	poll := env.Poll
	if poll <= 0 {
		poll = 2 * time.Second
	}
	deadline := time.Now().Add(e.timeout(env))
	for {
		problem := c.RunSteps(ctx, env.Kube, ns, nil)
		if problem.Message == "" {
			return nil
		}
		if problem.Final {
			return &FailedError{Lines: []string{problem.Message}}
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			failed := &FailedError{Lines: []string{problem.Message}}
			if ctx.Err() == nil {
				failed.Details = env.Kube.Diagnose(ctx, ns)
			}
			return failed
		}
		select {
		case <-ctx.Done():
		case <-time.After(poll):
		}
	}
}

// applyProblems keeps the lines of a failed `kubectl apply` that say what was
// rejected, falling back to all of them.
func applyProblems(out string) []string {
	var errs, all []string
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l == "" {
			continue
		}
		all = append(all, l)
		if strings.Contains(l, "rror") {
			errs = append(errs, l)
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return all
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
