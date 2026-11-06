package check

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"cel.dev/cel-go/cel"
)

// Kube is how steps read the cluster: kubectl, already pinned to the k8slings
// cluster by the caller. A step only ever reads.
type Kube interface {
	Kubectl(ctx context.Context, args ...string) (string, error)
}

// Step is one check against the live cluster. Exactly one field is set. All
// steps are read-only and idempotent, because the runner polls them.
type Step struct {
	Rollout   string         `json:"rollout"` // deployment/web, statefulset/db, daemonset/agent
	Ready     *ReadyStep     `json:"ready"`
	Assert    *AssertStep    `json:"assert"`
	Endpoints *EndpointsStep `json:"endpoints"`
	HTTP      *HTTPStep      `json:"http"`
	Exec      *ExecStep      `json:"exec"`
	CanI      *CanIStep      `json:"can_i"`
	Logs      *LogsStep      `json:"logs"`
	// Now marks a step whose answer waiting cannot change: when it fails, the
	// run ends at once instead of polling until the timeout. Use it for what the
	// learner wrote (a field of the spec) and for what an apply decides at once
	// (RBAC), not for what the cluster has to catch up with.
	Now bool `json:"now"`
}

// ReadyStep waits for Pods to be Ready: one by name, or at least Min with a label selector.
type ReadyStep struct {
	Pod      string `json:"pod"`
	Selector string `json:"selector"`
	Min      int    `json:"min"` // with a selector; default 1
}

// AssertStep evaluates a CEL expression on live objects: Resource ("kind/name")
// gives `object`; Resources (a kind, with an optional Selector) gives `objects`.
type AssertStep struct {
	Resource  string `json:"resource"`
	Resources string `json:"resources"`
	Selector  string `json:"selector"`
	Expr      string `json:"expr"`
	Message   string `json:"message"`
	// Now is the same as setting now on the step: kept so that
	// `assert: {..., now: true}` keeps working.
	Now bool `json:"now"`
}

// EndpointsStep wants at least Min ready endpoints behind a Service, and
// optionally that they forward to Port.
type EndpointsStep struct {
	Service string `json:"service"`
	Min     int    `json:"min"` // default 1
	Port    int    `json:"port"`
}

// HTTPStep requests a path of a Service through the API server's service proxy
// (no port-forward, no host port) and wants a 2xx answer. The proxy bypasses
// NetworkPolicy, so use an Exec step from a Pod to test those.
type HTTPStep struct {
	Service      string `json:"service"`
	Port         int    `json:"port"`
	Path         string `json:"path"`
	BodyContains string `json:"body_contains"`
}

// ExecStep runs a command in a running Pod. With OK false it wants the command
// to fail (a blocked connection, say); the Pod must be Running first, so a Pod
// that has not started cannot pass for "blocked".
type ExecStep struct {
	Pod       string   `json:"pod"`
	Container string   `json:"container"`
	Args      []string `json:"args"`
	OK        *bool    `json:"ok"` // default true
	Contains  string   `json:"contains"`
}

// CanIStep asks `kubectl auth can-i` as a ServiceAccount of the exercise
// namespace. Resource may name a subresource, such as pods/log.
type CanIStep struct {
	As        string `json:"as"` // the name of the ServiceAccount
	Verb      string `json:"verb"`
	Resource  string `json:"resource"`
	Namespace string `json:"namespace"` // default: the exercise namespace
	OK        *bool  `json:"ok"`        // default true
}

// LogsStep wants some text in the logs of a Pod.
type LogsStep struct {
	Pod       string `json:"pod"`
	Container string `json:"container"`
	Contains  string `json:"contains"`
}

type compiledStep struct {
	Step
	prog cel.Program // for Assert
}

func (s Step) kinds() int {
	n := 0
	for _, set := range []bool{s.Rollout != "", s.Ready != nil, s.Assert != nil, s.Endpoints != nil, s.HTTP != nil, s.Exec != nil, s.CanI != nil, s.Logs != nil} {
		if set {
			n++
		}
	}
	return n
}

func (c *Compiled) compileSteps() error {
	for i, s := range c.File.Steps {
		bad := func(format string, args ...any) error {
			return fmt.Errorf("check.yaml: steps[%d]: %s", i, fmt.Sprintf(format, args...))
		}
		if s.kinds() != 1 {
			return bad("a step has exactly one of rollout, ready, assert, endpoints, http, exec, can_i, logs")
		}
		cs := compiledStep{Step: s}
		switch {
		case s.Rollout != "":
			if kind, name, ok := strings.Cut(s.Rollout, "/"); !ok || name == "" || !strings.Contains("deployment statefulset daemonset", kind) {
				return bad("rollout wants deployment/<name>, statefulset/<name> or daemonset/<name>")
			}
		case s.Ready != nil:
			if (s.Ready.Pod == "") == (s.Ready.Selector == "") {
				return bad("ready wants a pod or a selector")
			}
		case s.Assert != nil:
			a := s.Assert
			if (a.Resource == "") == (a.Resources == "") {
				return bad("assert wants resource (kind/name) or resources (a kind)")
			}
			if strings.TrimSpace(a.Message) == "" {
				return bad("assert needs a message")
			}
			ast, iss := c.env.Compile(a.Expr)
			if iss.Err() != nil {
				return bad("assert: %v", iss.Err())
			}
			if ast.OutputType() != cel.BoolType && ast.OutputType() != cel.DynType {
				return bad("assert must be a boolean expression")
			}
			prg, err := c.env.Program(ast)
			if err != nil {
				return err
			}
			cs.prog = prg
		case s.Endpoints != nil:
			if s.Endpoints.Service == "" {
				return bad("endpoints needs a service")
			}
		case s.HTTP != nil:
			if s.HTTP.Service == "" || s.HTTP.Port == 0 {
				return bad("http needs a service and a port")
			}
		case s.Exec != nil:
			if s.Exec.Pod == "" || len(s.Exec.Args) == 0 {
				return bad("exec needs a pod and args")
			}
		case s.CanI != nil:
			if s.CanI.As == "" || s.CanI.Verb == "" || s.CanI.Resource == "" {
				return bad("can_i needs as, verb and resource")
			}
		case s.Logs != nil:
			if s.Logs.Pod == "" || s.Logs.Contains == "" {
				return bad("logs needs a pod and contains")
			}
		}
		c.steps = append(c.steps, cs)
	}
	return nil
}

// HasSteps reports whether the check has cluster steps.
func (c *Compiled) HasSteps() bool { return len(c.steps) > 0 }

// Problem is why the steps do not hold (yet).
type Problem struct {
	Message string
	// Final means waiting will not help: the run should end now.
	Final bool
}

// RunSteps evaluates the steps in order in the namespace ns and returns the
// problem of the first one that does not hold, or the zero Problem when all of
// them do. It only reads from the cluster.
func (c *Compiled) RunSteps(ctx context.Context, kube Kube, ns string) Problem {
	for _, s := range c.steps {
		if msg := s.run(ctx, kube, ns); msg != "" {
			return Problem{Message: msg, Final: s.Now || (s.Assert != nil && s.Assert.Now)}
		}
	}
	return Problem{}
}

func (s compiledStep) run(ctx context.Context, kube Kube, ns string) string {
	switch {
	case s.Rollout != "":
		return stepRollout(ctx, kube, ns, s.Rollout)
	case s.Ready != nil:
		return stepReady(ctx, kube, ns, *s.Ready)
	case s.Assert != nil:
		return s.stepAssert(ctx, kube, ns)
	case s.Endpoints != nil:
		return stepEndpoints(ctx, kube, ns, *s.Endpoints)
	case s.HTTP != nil:
		return stepHTTP(ctx, kube, ns, *s.HTTP)
	case s.Exec != nil:
		return stepExec(ctx, kube, ns, *s.Exec)
	case s.CanI != nil:
		return stepCanI(ctx, kube, ns, *s.CanI)
	case s.Logs != nil:
		return stepLogs(ctx, kube, ns, *s.Logs)
	}
	return ""
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}

// getJSON reads one object, or a list, as parsed JSON.
func getJSON(ctx context.Context, kube Kube, args ...string) (map[string]any, string) {
	out, err := kube.Kubectl(ctx, append([]string{"get"}, append(args, "-o", "json")...)...)
	if err != nil {
		return nil, firstLine(out)
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(out), &obj); err != nil {
		return nil, "cannot read the answer of kubectl: " + err.Error()
	}
	return obj, ""
}

func num(m map[string]any, path ...string) int {
	var cur any = m
	for _, p := range path {
		mm, ok := cur.(map[string]any)
		if !ok {
			return 0
		}
		cur = mm[p]
	}
	f, _ := cur.(float64)
	return int(f)
}

func str(m map[string]any, path ...string) string {
	var cur any = m
	for _, p := range path {
		mm, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = mm[p]
	}
	s, _ := cur.(string)
	return s
}

func stepRollout(ctx context.Context, kube Kube, ns, res string) string {
	kind, name, _ := strings.Cut(res, "/")
	obj, problem := getJSON(ctx, kube, res, "-n", ns)
	if obj == nil {
		return fmt.Sprintf("%s %s: %s", kind, name, problem)
	}
	switch kind {
	case "deployment":
		want := 1
		spec, _ := obj["spec"].(map[string]any)
		if _, set := spec["replicas"]; set {
			want = num(obj, "spec", "replicas")
		}
		if num(obj, "metadata", "generation") != num(obj, "status", "observedGeneration") {
			return fmt.Sprintf("Deployment %s has not seen your latest change yet.", name)
		}
		if got := num(obj, "status", "updatedReplicas"); got != want {
			return fmt.Sprintf("Deployment %s is rolling out: %d of %d replicas updated.", name, got, want)
		}
		if got := num(obj, "status", "availableReplicas"); got != want {
			return fmt.Sprintf("Deployment %s: %d of %d replicas are available.", name, got, want)
		}
		if got := num(obj, "status", "replicas"); got != want {
			return fmt.Sprintf("Deployment %s still has %d replicas, want %d.", name, got, want)
		}
	case "statefulset":
		want := num(obj, "spec", "replicas")
		if got := num(obj, "status", "readyReplicas"); got != want {
			return fmt.Sprintf("StatefulSet %s: %d of %d replicas are ready.", name, got, want)
		}
		if got := num(obj, "status", "updatedReplicas"); got != want {
			return fmt.Sprintf("StatefulSet %s is rolling out: %d of %d replicas updated.", name, got, want)
		}
	case "daemonset":
		want := num(obj, "status", "desiredNumberScheduled")
		if want == 0 {
			return fmt.Sprintf("DaemonSet %s is not scheduled on any node yet.", name)
		}
		if got := num(obj, "status", "numberReady"); got != want {
			return fmt.Sprintf("DaemonSet %s: %d of %d Pods are ready.", name, got, want)
		}
		if got := num(obj, "status", "updatedNumberScheduled"); got != want {
			return fmt.Sprintf("DaemonSet %s is rolling out: %d of %d Pods updated.", name, got, want)
		}
	}
	return ""
}

func podReady(pod map[string]any) bool {
	status, _ := pod["status"].(map[string]any)
	conds, _ := status["conditions"].([]any)
	for _, c := range conds {
		m, _ := c.(map[string]any)
		if m["type"] == "Ready" {
			return m["status"] == "True"
		}
	}
	return false
}

// podProblem describes why a Pod is not Ready, in a few words.
func podProblem(pod map[string]any) string {
	phase := str(pod, "status", "phase")
	if statuses, _ := pod["status"].(map[string]any)["containerStatuses"].([]any); len(statuses) > 0 {
		for _, cs := range statuses {
			m, _ := cs.(map[string]any)
			if reason := str(m, "state", "waiting", "reason"); reason != "" {
				return phase + ", " + reason
			}
			if reason := str(m, "state", "terminated", "reason"); reason != "" {
				return phase + ", " + reason
			}
		}
	}
	return phase
}

func stepReady(ctx context.Context, kube Kube, ns string, r ReadyStep) string {
	if r.Pod != "" {
		pod, problem := getJSON(ctx, kube, "pod/"+r.Pod, "-n", ns)
		if pod == nil {
			return fmt.Sprintf("Pod %s: %s", r.Pod, problem)
		}
		if !podReady(pod) {
			return fmt.Sprintf("Pod %s is not Ready yet (%s).", r.Pod, podProblem(pod))
		}
		return ""
	}
	list, problem := getJSON(ctx, kube, "pods", "-n", ns, "-l", r.Selector)
	if list == nil {
		return "Pods " + r.Selector + ": " + problem
	}
	items, _ := list["items"].([]any)
	min := max(r.Min, 1)
	ready, notReady := 0, ""
	for _, it := range items {
		pod, _ := it.(map[string]any)
		if podReady(pod) {
			ready++
		} else if notReady == "" {
			notReady = fmt.Sprintf("%s (%s)", str(pod, "metadata", "name"), podProblem(pod))
		}
	}
	if ready < min {
		msg := fmt.Sprintf("%d of %d expected Pods with %s are Ready", ready, min, r.Selector)
		if notReady != "" {
			msg += "; " + notReady + " is not"
		}
		return msg + "."
	}
	return ""
}

func (s compiledStep) stepAssert(ctx context.Context, kube Kube, ns string) string {
	a := s.Assert
	vars := map[string]any{"object": map[string]any{}, "objects": []any{}}
	if a.Resource != "" {
		obj, problem := getJSON(ctx, kube, a.Resource, "-n", ns)
		if obj == nil {
			return fmt.Sprintf("%s: %s", a.Resource, problem)
		}
		vars["object"] = obj
	} else {
		args := []string{a.Resources, "-n", ns}
		if a.Selector != "" {
			args = append(args, "-l", a.Selector)
		}
		list, problem := getJSON(ctx, kube, args...)
		if list == nil {
			return fmt.Sprintf("%s: %s", a.Resources, problem)
		}
		items, _ := list["items"].([]any)
		vars["objects"] = items
	}
	out, _, err := s.prog.Eval(vars)
	if err != nil || out.Value() != true {
		return a.Message
	}
	return ""
}

func stepEndpoints(ctx context.Context, kube Kube, ns string, e EndpointsStep) string {
	list, problem := getJSON(ctx, kube, "endpointslices", "-n", ns, "-l", "kubernetes.io/service-name="+e.Service)
	if list == nil {
		return fmt.Sprintf("Endpoints of Service %s: %s", e.Service, problem)
	}
	ready, onPort := 0, e.Port == 0
	items, _ := list["items"].([]any)
	for _, it := range items {
		slice, _ := it.(map[string]any)
		eps, _ := slice["endpoints"].([]any)
		for _, ep := range eps {
			m, _ := ep.(map[string]any)
			conds, _ := m["conditions"].(map[string]any)
			if conds["ready"] == true {
				ready++
			}
		}
		ports, _ := slice["ports"].([]any)
		for _, p := range ports {
			pm, _ := p.(map[string]any)
			if num(pm, "port") == e.Port {
				onPort = true
			}
		}
	}
	if ready < max(e.Min, 1) {
		return fmt.Sprintf("Service %s has %d ready endpoints, want at least %d: its selector must match Pods that are Ready.", e.Service, ready, max(e.Min, 1))
	}
	if !onPort {
		return fmt.Sprintf("Service %s does not forward to port %d.", e.Service, e.Port)
	}
	return ""
}

func stepHTTP(ctx context.Context, kube Kube, ns string, h HTTPStep) string {
	path := "/" + strings.TrimPrefix(h.Path, "/")
	raw := fmt.Sprintf("/api/v1/namespaces/%s/services/%s:%d/proxy%s", ns, h.Service, h.Port, path)
	out, err := kube.Kubectl(ctx, "get", "--raw", raw)
	if err != nil {
		return fmt.Sprintf("GET %s on Service %s:%d fails: %s", path, h.Service, h.Port, firstLine(out))
	}
	if h.BodyContains != "" && !strings.Contains(out, h.BodyContains) {
		return fmt.Sprintf("GET %s on Service %s:%d does not contain %q.", path, h.Service, h.Port, h.BodyContains)
	}
	return ""
}

func stepExec(ctx context.Context, kube Kube, ns string, e ExecStep) string {
	phase, err := kube.Kubectl(ctx, "get", "pod", e.Pod, "-n", ns, "-o", "jsonpath={.status.phase}")
	if err != nil {
		return fmt.Sprintf("Pod %s: %s", e.Pod, firstLine(phase))
	}
	if phase != "Running" {
		return fmt.Sprintf("Pod %s is %s, not Running yet.", e.Pod, phase)
	}
	args := []string{"exec", e.Pod, "-n", ns}
	if e.Container != "" {
		args = append(args, "-c", e.Container)
	}
	out, err := kube.Kubectl(ctx, append(append(args, "--"), e.Args...)...)
	want := e.OK == nil || *e.OK
	switch {
	case want && err != nil:
		return fmt.Sprintf("In Pod %s, `%s` fails: %s", e.Pod, strings.Join(e.Args, " "), firstLine(out))
	case want && e.Contains != "" && !strings.Contains(out, e.Contains):
		return fmt.Sprintf("In Pod %s, `%s` does not print %q.", e.Pod, strings.Join(e.Args, " "), e.Contains)
	case !want && err == nil:
		return fmt.Sprintf("In Pod %s, `%s` works, but it should be blocked.", e.Pod, strings.Join(e.Args, " "))
	}
	return ""
}

func stepCanI(ctx context.Context, kube Kube, ns string, c CanIStep) string {
	target := c.Namespace
	if target == "" {
		target = ns
	}
	who := fmt.Sprintf("system:serviceaccount:%s:%s", ns, c.As)
	// kubectl reads `pods/log` as "the Pod called log", not as the log
	// subresource, so a subresource must be given with --subresource. Without
	// this, a check for pods/log answers "yes" for the wrong thing.
	resource, subresource, _ := strings.Cut(c.Resource, "/")
	args := []string{"auth", "can-i", c.Verb, resource, "-n", target, "--as", who}
	if subresource != "" {
		args = append(args, "--subresource="+subresource)
	}
	out, _ := kube.Kubectl(ctx, args...)
	got := strings.TrimSpace(out)
	if got != "yes" && got != "no" {
		return fmt.Sprintf("cannot ask what %s may do: %s", c.As, firstLine(out))
	}
	want := c.OK == nil || *c.OK
	switch {
	case want && got != "yes":
		return fmt.Sprintf("ServiceAccount %s cannot %s %s in %s yet.", c.As, c.Verb, c.Resource, target)
	case !want && got != "no":
		return fmt.Sprintf("ServiceAccount %s must NOT be able to %s %s in %s (least privilege).", c.As, c.Verb, c.Resource, target)
	}
	return ""
}

func stepLogs(ctx context.Context, kube Kube, ns string, l LogsStep) string {
	args := []string{"logs", l.Pod, "-n", ns}
	if l.Container != "" {
		args = append(args, "-c", l.Container)
	}
	out, err := kube.Kubectl(ctx, args...)
	if err != nil {
		return fmt.Sprintf("cannot read the logs of Pod %s yet: %s", l.Pod, firstLine(out))
	}
	if !strings.Contains(out, l.Contains) {
		return fmt.Sprintf("The logs of Pod %s should contain %q, they say %q.", l.Pod, l.Contains, firstLine(out))
	}
	return ""
}
