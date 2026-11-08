package exercises

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cleonte/k8slings/k8slings/check"
	"github.com/cleonte/k8slings/k8slings/cluster"
	"github.com/cleonte/k8slings/k8slings/query"
)

// How many lines of each output a failure shows. The outputs are compared in
// full; this only limits what is printed.
const queryShownLines = 20

// queryLocks serialises the runs of one exercise in this process: the variant
// changes the namespace for a while, and `watch` and a check of everything must
// not both look at it then.
var queryLocks sync.Map // exercise name -> *sync.Mutex

func lockQuery(name string) func() {
	m, _ := queryLocks.LoadOrStore(name, &sync.Mutex{})
	m.(*sync.Mutex).Lock()
	return m.(*sync.Mutex).Unlock
}

// runQuery checks the answer of a query exercise: the command the learner wrote
// is parsed and checked by the query package (it never reaches a shell), run
// against the seeded state, and its output is compared with what the reference
// answer prints against the same state. If the exercise has a variant, the same
// is done once more with extra objects in the namespace, which is what stops an
// answer that has the names and values of the first state typed into it. A run
// only succeeds, and so only records a pass, after both.
func (e Exercise) runQuery(ctx context.Context, env *Env, c *check.Compiled, src []byte) (Result, error) {
	spec := c.Query
	if spec == nil {
		return Result{}, fmt.Errorf("%s: check.yaml has no query section", e.Name)
	}
	if env.Kube == nil {
		return Result{}, &SkippedError{fmt.Sprintf("%s cannot run kubectl here", e.Name)}
	}
	defer lockQuery(e.Name)()

	opt := query.Options{AllNamespaces: spec.AllNamespaces}
	if !e.Offline {
		opt.Namespace = e.Namespace()
	}
	cmd, err := query.Parse(string(src), opt)
	if err != nil {
		var refused *query.RefusedError
		if errors.As(err, &refused) {
			return Result{}, &FailedError{Lines: []string{refused.Msg}}
		}
		return Result{}, err
	}
	if e.Offline && !cmd.Offline {
		return Result{}, &FailedError{Lines: []string{"This exercise has no cluster: only kubectl create ... --dry-run=client works here."}}
	}
	// What the command breaks of the exercise's own rules (it names a Pod, it has
	// no --sort-by) is reported with, and after, what its output got wrong: the
	// learner sees the output blocks first, which usually say more.
	rules := c.RunCommand(cmd.Verb, cmd.Positional, cmd.Flags)

	if e.hasSetup(env) {
		if !env.Kube.Seeded(ctx, e.Namespace(), e.Name) || !hasSeed(e.Name) {
			if noSeed(ctx) {
				return Result{}, &SkippedError{fmt.Sprintf("%s has not been started: run it to set up its cluster state", e.Name)}
			}
			if err := e.seedQuery(ctx, env, c); err != nil {
				return Result{}, err
			}
		} else if err := e.waitSteps(ctx, env, c, false); err != nil {
			return Result{}, err
		}
	}

	if err := e.compareQuery(ctx, env, c, cmd, opt); err != nil {
		var failed *FailedError
		if errors.As(err, &failed) {
			failed.Lines = append(failed.Lines, rules...)
		}
		return Result{}, err
	}
	if len(rules) > 0 {
		return Result{}, &FailedError{Lines: rules}
	}

	if variant, err := fs.ReadFile(env.Embedded, path.Join(e.Path, VariantFile)); err == nil && e.hasSetup(env) {
		if err := e.runVariant(ctx, env, c, cmd, opt, variant); err != nil {
			return Result{}, err
		}
	}
	return Result{Exercise: e}, nil
}

// hasSetup says whether the exercise has a cluster state to set up. An exercise
// can use the cluster without one (kubectl explain asks the API server only).
func (e Exercise) hasSetup(env *Env) bool {
	_, err := fs.Stat(env.Embedded, e.SetupPath())
	return err == nil
}

// seedQuery brings the namespace of a query exercise to its state and waits until
// the steps of the check say it is ready.
func (e Exercise) seedQuery(ctx context.Context, env *Env, c *check.Compiled) error {
	if err := e.seed(ctx, env, c); err != nil {
		return err
	}
	return e.waitSteps(ctx, env, c, false)
}

// waitSteps polls the steps of the check (and, for the variant, its own) until
// they hold or the exercise's time is up: the seeded state may need a moment
// (a Pod that has to crash once).
func (e Exercise) waitSteps(ctx context.Context, env *Env, c *check.Compiled, variant bool) error {
	poll := env.Poll
	if poll <= 0 {
		poll = time.Second
	}
	deadline := time.Now().Add(e.timeout(env))
	for {
		p := c.RunSteps(ctx, env.Kube, e.Namespace(), nil)
		if p.Message == "" && variant {
			p = c.RunVariantSteps(ctx, env.Kube, e.Namespace())
		}
		if p.Message == "" {
			return nil
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("the state of %s did not become ready: %s (try `k8slings reset %s`)", e.Name, p.Message, e.Name)
		}
		select {
		case <-ctx.Done():
		case <-time.After(poll):
		}
	}
}

// compareQuery runs the learner's command, and either compares its output with
// the reference's or checks it as a manifest.
func (e Exercise) compareQuery(ctx context.Context, env *Env, c *check.Compiled, cmd *query.Command, opt query.Options) error {
	got, failed, err := e.runCommand(ctx, env, cmd)
	if failed != nil || err != nil {
		if failed != nil {
			return failed
		}
		return err
	}
	if c.Query.Compare == check.CompareManifest {
		return e.checkManifest(env, c, got.Stdout)
	}
	want, err := e.reference(ctx, env, opt)
	if err != nil {
		return err
	}
	g, w := normalize(got.Stdout, c.Query.Sort), normalize(want, c.Query.Sort)
	if g == w {
		return nil
	}
	return &FailedError{
		Lines:   []string{"The output is not the one this exercise asks for."},
		Details: sideBySide(g, w, got.Stderr),
	}
}

// runCommand runs a parsed command. A problem of the learner's command comes back
// as a *FailedError, a problem of the machine (no kubectl) as a *SkippedError.
func (e Exercise) runCommand(ctx context.Context, env *Env, cmd *query.Command) (cluster.QueryResult, *FailedError, error) {
	res, err := env.Kube.Query(ctx, cmd.Args, e.Offline || cmd.Offline)
	switch {
	case errors.Is(err, cluster.ErrNoKubectl):
		return res, nil, &SkippedError{fmt.Sprintf("%s needs kubectl, which is not installed (see `k8slings doctor`)", e.Name)}
	case ctx.Err() != nil:
		return res, nil, ctx.Err()
	case res.TimedOut:
		return res, &FailedError{Lines: []string{fmt.Sprintf("The command took more than %d seconds and was stopped.", int(cluster.QueryTimeout/time.Second))}}, nil
	case res.Truncated:
		return res, &FailedError{Lines: []string{"The command printed more than 1 MB."}}, nil
	case err != nil:
		var forbidden cluster.ErrForbiddenFlag
		if errors.As(err, &forbidden) {
			return res, &FailedError{Lines: []string{err.Error()}}, nil
		}
		lines := nonEmptyLines(res.Stderr)
		if len(lines) == 0 {
			lines = []string{err.Error()}
		}
		return res, &FailedError{Lines: []string{"kubectl failed: " + lines[0]}, Details: head(lines[1:])}, nil
	}
	return res, nil, nil
}

// reference is what the reference answer of the exercise prints now. A reference
// that fails is a bug of the exercise, not of the learner.
func (e Exercise) reference(ctx context.Context, env *Env, opt query.Options) (string, error) {
	if env.Solutions == nil {
		return "", fmt.Errorf("%s: no solutions are available to compare with", e.Name)
	}
	src, err := fs.ReadFile(env.Solutions, e.SolutionPath())
	if err != nil {
		return "", fmt.Errorf("%s has no reference answer: %w", e.Name, err)
	}
	cmd, err := query.Parse(string(src), opt)
	if err != nil {
		return "", fmt.Errorf("the reference answer of %s is refused: %w", e.Name, err)
	}
	res, failed, err := e.runCommand(ctx, env, cmd)
	if err != nil {
		return "", err
	}
	if failed != nil {
		return "", fmt.Errorf("the reference answer of %s failed (a bug of the exercise): %s", e.Name, failed.Lines[0])
	}
	return res.Stdout, nil
}

// checkManifest reads the output as YAML and runs the schema and the assertions
// of the check on it.
func (e Exercise) checkManifest(env *Env, c *check.Compiled, out string) error {
	objs, err := ParseObjects([]byte(out))
	if err != nil {
		return &FailedError{Lines: []string{"The output is not YAML: " + err.Error()}}
	}
	if len(objs) == 0 {
		return &FailedError{Lines: []string{"The command printed no Kubernetes object: add -o yaml."}}
	}
	if c.Schema == "strict" {
		var problems []string
		for _, o := range objs {
			ps, err := env.Validator.Validate(o)
			if err != nil {
				return fmt.Errorf("%s: %w", e.Name, err)
			}
			for _, p := range ps {
				problems = append(problems, fmt.Sprintf("%s %s: %s", o["kind"], objectName(o), p))
			}
		}
		if len(problems) > 0 {
			return &FailedError{Lines: problems}
		}
	}
	if failures := c.Run(objs); len(failures) > 0 {
		return &FailedError{Lines: failures}
	}
	return nil
}

// runVariant adds the objects of setup-variant.yaml, waits for them, runs the
// learner's command and the reference once more, and takes the objects away
// again. An answer with the names or values of the first state typed in does not
// survive it.
func (e Exercise) runVariant(ctx context.Context, env *Env, c *check.Compiled, cmd *query.Command, opt query.Options, variant []byte) (err error) {
	ns := e.Namespace()
	// Whatever happens, the namespace gets its state back, even when this run is
	// cancelled (a save in `watch` starts the next one).
	defer func() {
		clean, cancel := context.WithTimeout(context.WithoutCancel(ctx), 90*time.Second)
		defer cancel()
		if derr := env.Kube.Delete(clean, ns, variant); derr != nil && err == nil {
			err = derr
		}
	}()
	// A variant left behind by a run that was killed would be in the way.
	if err := env.Kube.Delete(ctx, ns, variant); err != nil {
		return err
	}
	if out, aerr := env.Kube.Apply(ctx, ns, variant); aerr != nil {
		return fmt.Errorf("cannot apply the variant of %s: %s", e.Name, firstLineOf(out))
	}
	if werr := e.waitSteps(ctx, env, c, true); werr != nil {
		return werr
	}
	if cerr := e.compareQuery(ctx, env, c, cmd, opt); cerr != nil {
		var failed *FailedError
		if errors.As(cerr, &failed) {
			return &FailedError{
				Lines:   []string{"The output was right for the objects you see now, but not when other objects are in the namespace. The command must find what is asked for, not repeat the names and values of today."},
				Details: failed.Details,
			}
		}
		return cerr
	}
	return nil
}

// normalize makes two outputs comparable: it drops what a terminal would not
// show (carriage returns, trailing spaces and blank lines at the end) and, if
// asked, sorts the lines.
func normalize(s string, sortLines bool) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t\r")
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if sortLines {
		slices.Sort(lines)
	}
	return strings.Join(lines, "\n")
}

// sideBySide is the details of a failed comparison: what the command printed and
// what was expected (the expected output is part of the task, not a spoiler),
// each cut to the first lines.
func sideBySide(got, want, stderr string) []string {
	out := []string{"Your command printed:"}
	out = append(out, indent(shown(got))...)
	if lines := nonEmptyLines(stderr); len(lines) > 0 {
		out = append(out, "and kubectl said:")
		out = append(out, indent(head(lines))...)
	}
	out = append(out, "", "Expected:")
	out = append(out, indent(shown(want))...)
	return out
}

func shown(s string) []string {
	if s == "" {
		return []string{"(nothing)"}
	}
	return head(strings.Split(s, "\n"))
}

func head(lines []string) []string {
	if len(lines) <= queryShownLines {
		return lines
	}
	return append(slices.Clone(lines[:queryShownLines]), fmt.Sprintf("... %d more lines", len(lines)-queryShownLines))
}

func indent(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = "  " + l
	}
	return out
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func firstLineOf(s string) string {
	if l := nonEmptyLines(s); len(l) > 0 {
		return l[0]
	}
	return s
}
