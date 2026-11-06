package check

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeKube answers kubectl calls by the first rule whose words all appear in
// the command line.
type fakeKube struct {
	rules []rule
	calls []string
}

type rule struct {
	words []string
	out   string
	err   bool
}

func (f *fakeKube) Kubectl(_ context.Context, args ...string) (string, error) {
	line := strings.Join(args, " ")
	f.calls = append(f.calls, line)
	for _, r := range f.rules {
		match := true
		for _, w := range r.words {
			if !strings.Contains(line, w) {
				match = false
			}
		}
		if match {
			if r.err {
				return r.out, errors.New("exit status 1")
			}
			return r.out, nil
		}
	}
	return "unexpected: " + line, errors.New("exit status 1")
}

func steps(t *testing.T, yaml string) *Compiled {
	t.Helper()
	c, err := Load([]byte("schema: off\nsteps:\n" + yaml))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func run(c *Compiled, k *fakeKube) string {
	return c.RunSteps(context.Background(), k, "ns1", nil).Message
}

const deployJSON = `{"metadata":{"generation":2},"spec":{"replicas":3},"status":{"observedGeneration":%OBS%,"replicas":3,"updatedReplicas":%UPD%,"availableReplicas":%AVAIL%}}`

func deploy(obs, upd, avail string) string {
	return strings.NewReplacer("%OBS%", obs, "%UPD%", upd, "%AVAIL%", avail).Replace(deployJSON)
}

func TestRolloutWaitsForGenerationUpdatedAndAvailable(t *testing.T) {
	c := steps(t, "  - rollout: deployment/shop\n")
	for name, tc := range map[string]struct{ json, want string }{
		"not observed": {deploy("1", "3", "3"), "not seen your latest change"},
		"updating":     {deploy("2", "1", "3"), "1 of 3 replicas updated"},
		"unavailable":  {deploy("2", "3", "2"), "2 of 3 replicas are available"},
		"done":         {deploy("2", "3", "3"), ""},
	} {
		got := run(c, &fakeKube{rules: []rule{{words: []string{"get", "deployment/shop", "-n ns1", "-o json"}, out: tc.json}}})
		if (tc.want == "" && got != "") || !strings.Contains(got, tc.want) {
			t.Errorf("%s: got %q, want %q", name, got, tc.want)
		}
	}
}

func TestStepsReportOnlyTheFirstProblemAndOnlyRead(t *testing.T) {
	c := steps(t, "  - rollout: deployment/shop\n  - logs: {pod: x, contains: y}\n")
	k := &fakeKube{rules: []rule{{words: []string{"deployment/shop"}, out: deploy("2", "1", "3")}}}
	if got := run(c, k); !strings.Contains(got, "rolling out") {
		t.Fatalf("got %q", got)
	}
	for _, call := range k.calls {
		if !strings.HasPrefix(call, "get ") {
			t.Fatalf("a step wrote to or ran something in the cluster: %s", call)
		}
	}
	if len(k.calls) != 1 {
		t.Fatalf("later steps should not run after a failure: %v", k.calls)
	}
}

func TestReadyByNameAndBySelector(t *testing.T) {
	pod := func(name, ready, waiting string) string {
		w := ""
		if waiting != "" {
			w = `,"containerStatuses":[{"state":{"waiting":{"reason":"` + waiting + `"}}}]`
		}
		return `{"metadata":{"name":"` + name + `"},"status":{"phase":"Running","conditions":[{"type":"Ready","status":"` + ready + `"}]` + w + `}}`
	}
	c := steps(t, "  - ready: {pod: site}\n")
	if got := run(c, &fakeKube{rules: []rule{{words: []string{"pod/site"}, out: pod("site", "False", "CrashLoopBackOff")}}}); !strings.Contains(got, "not Ready yet (Running, CrashLoopBackOff)") {
		t.Fatalf("got %q", got)
	}
	if got := run(c, &fakeKube{rules: []rule{{words: []string{"pod/site"}, out: pod("site", "True", "")}}}); got != "" {
		t.Fatalf("got %q", got)
	}

	c = steps(t, "  - ready: {selector: app=web, min: 2}\n")
	list := `{"items":[` + pod("a", "True", "") + `,` + pod("b", "False", "ImagePullBackOff") + `]}`
	got := run(c, &fakeKube{rules: []rule{{words: []string{"pods", "-l app=web"}, out: list}}})
	if !strings.Contains(got, "1 of 2 expected Pods") || !strings.Contains(got, "b (Running, ImagePullBackOff)") {
		t.Fatalf("got %q", got)
	}
}

func TestAssertOnOneObjectAndOnAList(t *testing.T) {
	c := steps(t, `  - assert: {resource: deployment/api, expr: "object.spec.replicas == 3", message: "three replicas"}
  - assert: {resources: pods, selector: app=api, expr: "objects.size() == 2 && objects.all(p, p.status.phase == 'Running')", message: "two running pods"}
`)
	k := &fakeKube{rules: []rule{
		{words: []string{"deployment/api"}, out: `{"spec":{"replicas":3}}`},
		{words: []string{"pods", "-l app=api"}, out: `{"items":[{"status":{"phase":"Running"}},{"status":{"phase":"Pending"}}]}`},
	}}
	if got := run(c, k); got != "two running pods" {
		t.Fatalf("got %q", got)
	}
	k.rules[0].out = `{"spec":{"replicas":1}}`
	if got := run(c, k); got != "three replicas" {
		t.Fatalf("got %q", got)
	}
	k.rules[0].err = true
	k.rules[0].out = `Error from server (NotFound): deployments.apps "api" not found`
	if got := run(c, k); !strings.Contains(got, "deployment/api: Error from server (NotFound)") {
		t.Fatalf("got %q", got)
	}
}

func TestEndpoints(t *testing.T) {
	c := steps(t, "  - endpoints: {service: web, min: 2, port: 80}\n")
	slice := func(ready int, port int) string {
		eps := ""
		for i := 0; i < ready; i++ {
			eps += `{"conditions":{"ready":true}},`
		}
		eps += `{"conditions":{"ready":false}}`
		return `{"items":[{"endpoints":[` + eps + `],"ports":[{"port":` + string(rune('0'+port/10)) + string(rune('0'+port%10)) + `}]}]}`
	}
	rules := func(out string) *fakeKube {
		return &fakeKube{rules: []rule{{words: []string{"endpointslices", "kubernetes.io/service-name=web"}, out: out}}}
	}
	if got := run(c, rules(slice(1, 80))); !strings.Contains(got, "1 ready endpoints, want at least 2") {
		t.Fatalf("got %q", got)
	}
	if got := run(c, rules(slice(2, 80))); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := run(c, rules(slice(2, 81))); !strings.Contains(got, "does not forward to port 80") {
		t.Fatalf("got %q", got)
	}
}

func TestHTTPGoesThroughTheServiceProxy(t *testing.T) {
	c := steps(t, "  - http: {service: web, port: 80, path: /, body_contains: nginx}\n")
	k := &fakeKube{rules: []rule{{words: []string{"get --raw /api/v1/namespaces/ns1/services/web:80/proxy/"}, out: "<title>Welcome to nginx!</title>"}}}
	if got := run(c, k); got != "" {
		t.Fatalf("got %q", got)
	}
	k.rules[0].out = "hello"
	if got := run(c, k); !strings.Contains(got, `does not contain "nginx"`) {
		t.Fatalf("got %q", got)
	}
	k.rules[0].err, k.rules[0].out = true, "Error from server (ServiceUnavailable)"
	if got := run(c, k); !strings.Contains(got, "fails: Error from server") {
		t.Fatalf("got %q", got)
	}
}

func TestExecNeedsARunningPodAndCanExpectFailure(t *testing.T) {
	c := steps(t, `  - exec: {pod: client, args: [wget, -qO-, http://server]}
  - exec: {pod: intruder, args: [wget, -qO-, -T, "3", http://server], ok: false}
`)
	run1 := func(clientPhase, intruderPhase string, clientFails, intruderFails bool) string {
		return run(c, &fakeKube{rules: []rule{
			{words: []string{"get pod client"}, out: clientPhase},
			{words: []string{"get pod intruder"}, out: intruderPhase},
			{words: []string{"exec client"}, out: "page", err: clientFails},
			{words: []string{"exec intruder"}, out: "wget: download timed out", err: intruderFails},
		}})
	}
	if got := run1("Running", "Running", false, true); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := run1("Pending", "Running", false, true); !strings.Contains(got, "Pod client is Pending, not Running") {
		t.Fatalf("got %q", got)
	}
	if got := run1("Running", "Running", true, true); !strings.Contains(got, "`wget -qO- http://server` fails") {
		t.Fatalf("got %q", got)
	}
	if got := run1("Running", "Running", false, false); !strings.Contains(got, "should be blocked") {
		t.Fatalf("got %q", got)
	}
	// A Pod that has not started must not count as "blocked".
	if got := run1("Running", "Pending", false, true); !strings.Contains(got, "Pod intruder is Pending") {
		t.Fatalf("got %q", got)
	}
}

func TestCanI(t *testing.T) {
	c := steps(t, `  - can_i: {as: reader, verb: list, resource: pods}
  - can_i: {as: reader, verb: delete, resource: pods, ok: false}
`)
	k := &fakeKube{rules: []rule{
		{words: []string{"auth can-i list pods", "--as system:serviceaccount:ns1:reader"}, out: "yes"},
		{words: []string{"auth can-i delete pods"}, out: "yes"},
	}}
	if got := run(c, k); !strings.Contains(got, "must NOT be able to delete pods") {
		t.Fatalf("got %q", got)
	}
	k.rules[1].out = "no"
	if got := run(c, k); got != "" {
		t.Fatalf("got %q", got)
	}
}

// kubectl auth can-i get pods/log asks about the Pod called "log"; the log
// subresource needs --subresource. A check that forgot this passed a Role
// without any access to pod logs.
func TestCanIPassesSubresourcesAsSubresources(t *testing.T) {
	c := steps(t, "  - can_i: {as: deployer, verb: get, resource: pods/log}\n")
	k := &fakeKube{rules: []rule{{words: []string{"auth can-i get pods -n ns1 --as system:serviceaccount:ns1:deployer --subresource=log"}, out: "no"}}}
	if got := run(c, k); !strings.Contains(got, "cannot get pods/log") {
		t.Fatalf("got %q (calls: %v)", got, k.calls)
	}
	for _, call := range k.calls {
		if strings.Contains(call, "pods/log") {
			t.Fatalf("pods/log must not be passed to kubectl as one word: %s", call)
		}
	}
}

func TestLogs(t *testing.T) {
	c := steps(t, "  - logs: {pod: app, contains: level=debug}\n")
	k := &fakeKube{rules: []rule{{words: []string{"logs app -n ns1"}, out: "level=info"}}}
	if got := run(c, k); !strings.Contains(got, `should contain "level=debug"`) {
		t.Fatalf("got %q", got)
	}
	k.rules[0].out = "level=debug"
	if got := run(c, k); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestNowOnAnyStep(t *testing.T) {
	c := steps(t, "  - can_i: {as: reader, verb: list, resource: pods}\n    now: true\n")
	k := &fakeKube{rules: []rule{{words: []string{"auth can-i list pods"}, out: "no"}}}
	if p := c.RunSteps(context.Background(), k, "ns1", nil); !p.Final || !strings.Contains(p.Message, "cannot list pods") {
		t.Fatalf("now on a can_i step should be final: %+v", p)
	}
}

func TestLoadRejectsBadSteps(t *testing.T) {
	for name, yaml := range map[string]string{
		"two kinds":      "  - {rollout: deployment/a, logs: {pod: a, contains: b}}\n",
		"no kind":        "  - {}\n",
		"bad rollout":    "  - rollout: pod/a\n",
		"ready both":     "  - ready: {pod: a, selector: b=c}\n",
		"assert no msg":  "  - assert: {resource: pod/a, expr: 'true'}\n",
		"assert bad cel": "  - assert: {resource: pod/a, expr: 'object.', message: m}\n",
		"http no port":   "  - http: {service: a}\n",
		"exec no args":   "  - exec: {pod: a}\n",
		"can_i no verb":  "  - can_i: {as: a, resource: pods}\n",
		"unknown step":   "  - sleep: 5\n",
	} {
		if _, err := Load([]byte("schema: off\nsteps:\n" + yaml)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestNowAssertionsEndTheRunAtOnce(t *testing.T) {
	c := steps(t, `  - assert: {resource: deployment/api, expr: "object.spec.replicas == 3", message: "three replicas", now: true}
  - assert: {resource: deployment/api, expr: "object.status.ready == 3", message: "three ready"}
`)
	k := &fakeKube{rules: []rule{{words: []string{"deployment/api"}, out: `{"spec":{"replicas":1},"status":{"ready":0}}`}}}
	if p := c.RunSteps(context.Background(), k, "ns1", nil); p.Message != "three replicas" || !p.Final {
		t.Fatalf("an assertion about the spec should be final: %+v", p)
	}
	k.rules[0].out = `{"spec":{"replicas":3},"status":{"ready":0}}`
	if p := c.RunSteps(context.Background(), k, "ns1", nil); p.Message != "three ready" || p.Final {
		t.Fatalf("an assertion about the status is worth waiting for: %+v", p)
	}
}

type fakeSeeds map[string]string

func (f fakeSeeds) UID(resource string) (string, bool) { u, ok := f[resource]; return u, ok }

func TestUnchangedTellsTheSeededObjectFromANewOne(t *testing.T) {
	c := steps(t, "  - unchanged: deployment/web\n")
	k := &fakeKube{rules: []rule{{words: []string{"get deployment/web -n ns1", "jsonpath={.metadata.uid}"}, out: "uid-1"}}}
	run := func(seeds Seeds) string { return c.RunSteps(context.Background(), k, "ns1", seeds).Message }

	if got := run(fakeSeeds{"deployment/web": "uid-1"}); got != "" {
		t.Fatalf("the same object passes, got %q", got)
	}
	if got := run(fakeSeeds{"deployment/web": "uid-0"}); !strings.Contains(got, "deleted and created again") {
		t.Fatalf("a new object with the same name is not a repair, got %q", got)
	}
	// If k8slings cannot tell, it says so instead of waving it through.
	if got := run(fakeSeeds{}); !strings.Contains(got, "reset the exercise") {
		t.Fatalf("no recorded UID: %q", got)
	}
	if got := run(nil); !strings.Contains(got, "not been seeded") {
		t.Fatalf("no seeds at all: %q", got)
	}
	k.rules[0].err, k.rules[0].out = true, `Error from server (NotFound): deployments.apps "web" not found`
	if got := run(fakeSeeds{"deployment/web": "uid-1"}); !strings.Contains(got, "deployment/web: Error from server (NotFound)") {
		t.Fatalf("a missing object: %q", got)
	}
}

func TestUnchangedResourcesAreListedForTheSeeding(t *testing.T) {
	c := steps(t, "  - unchanged: deployment/web\n  - unchanged: service/web\n  - logs: {pod: a, contains: b}\n")
	got := c.UnchangedResources()
	if len(got) != 2 || got[0] != "deployment/web" || got[1] != "service/web" {
		t.Fatalf("got %v", got)
	}
}

func TestLoadRejectsABadUnchanged(t *testing.T) {
	if _, err := Load([]byte("schema: \"off\"\nsteps:\n  - unchanged: web\n")); err == nil {
		t.Fatal("unchanged wants kind/name")
	}
}
