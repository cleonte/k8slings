package exercises

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/cleonte/k8slings/k8slings/check"
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
	Nodes(ctx context.Context) (int, error)
	Diagnose(ctx context.Context, ns string) []string
}

// Env is what running an exercise needs: the embedded exercise tree (for the
// check.yaml files, which are never written to the workspace), the schemas and,
// for cluster exercises, the cluster.
type Env struct {
	Embedded  fs.FS // the k8slings.FS: info.toml and exercises/
	Validator *schema.Validator
	// Kube is nil when there is no cluster support; cluster exercises are then skipped.
	Kube Kube
	// Poll is how often a cluster check is repeated; default 2 seconds.
	Poll time.Duration
	// Timeout, when set, replaces the timeout of every cluster exercise. The
	// proof that an unsolved exercise fails uses a short one.
	Timeout time.Duration
}

// DefaultTimeout is how long a cluster exercise has to start working.
const DefaultTimeout = 60 * time.Second

// Availability looks at what this machine can run right now.
func (e *Env) Availability(ctx context.Context) Availability {
	if e == nil || e.Kube == nil || e.Kube.Verify(ctx) != nil {
		return Availability{}
	}
	n, _ := e.Kube.Nodes(ctx)
	return Availability{Cluster: true, Nodes: n}
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
	src, err := os.ReadFile(e.File())
	if err != nil {
		return Result{}, fmt.Errorf("cannot read %s: %w", e.File(), err)
	}
	return e.RunSource(ctx, env, src)
}

// RunSource checks src as if it were the content of the exercise file.
func (e Exercise) RunSource(ctx context.Context, env *Env, src []byte) (Result, error) {
	if e.Mode == ModeFix {
		return Result{}, &SkippedError{fmt.Sprintf("%s: %s", e.Name, e.SkipReason(Availability{}))}
	}
	if e.Mode == ModeCluster {
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
		problem := c.RunSteps(ctx, env.Kube, ns)
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
