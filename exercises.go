package main

import (
	"fmt"
	"strings"
	"time"
)

// Exercise describes one lesson. The learner-facing files (README.md,
// exercise.yaml, hint.md) live in exercises/<ID>/ and are embedded.
type Exercise struct {
	ID      string
	Title   string
	Timeout time.Duration // how long to wait for the cluster to converge
	Check   func(e *Env) error
}

// Namespace each exercise runs in. Isolated so cleanup is trivial.
func (ex *Exercise) Namespace() string {
	return "k8slings-" + strings.ReplaceAll(ex.ID, "_", "-")
}

// Env gives checks a small, readable API over kubectl.
type Env struct {
	NS string
}

// get returns a jsonpath value, e.g. e.get("pod/hello", "{.status.phase}").
func (e *Env) get(res, jsonpath string) (string, error) {
	out, err := kubectl("get", res, "-n", e.NS, "-o", "jsonpath="+jsonpath)
	if err != nil {
		return "", fmt.Errorf("%s not found in namespace %s", res, e.NS)
	}
	return out, nil
}

func (e *Env) expect(res, jsonpath, want, what string) error {
	got, err := e.get(res, jsonpath)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("%s: expected %q, got %q", what, want, got)
	}
	return nil
}

func (e *Env) logs(pod string) (string, error) {
	out, err := kubectl("logs", pod, "-n", e.NS)
	if err != nil {
		return "", fmt.Errorf("cannot read logs of %s yet: %s", pod, firstLine(out))
	}
	return out, nil
}

// exec runs a command inside a pod (read-only for the cluster state).
func (e *Env) exec(pod, container string, cmd ...string) (string, error) {
	args := []string{"exec", pod, "-n", e.NS}
	if container != "" {
		args = append(args, "-c", container)
	}
	return kubectl(append(args, append([]string{"--"}, cmd...)...)...)
}

// canI reports the answer of `kubectl auth can-i` for a ServiceAccount of the
// exercise namespace: "yes" or "no".
func (e *Env) canI(sa, verb, resource, ns string) string {
	who := fmt.Sprintf("system:serviceaccount:%s:%s", e.NS, sa)
	out, _ := kubectl("auth", "can-i", verb, resource, "-n", ns, "--as", who)
	return strings.TrimSpace(out)
}

func (e *Env) podRunning(pod string) error {
	phase, err := e.get("pod/"+pod, "{.status.phase}")
	if err != nil {
		return err
	}
	if phase != "Running" {
		reason, _ := e.get("pod/"+pod, "{.status.containerStatuses[0].state.waiting.reason}")
		if reason != "" {
			return fmt.Errorf("pod %s is %s (%s)", pod, phase, reason)
		}
		return fmt.Errorf("pod %s is %s, want Running", pod, phase)
	}
	return nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

var registry = []*Exercise{
	{
		ID: "01_pod_basics", Title: "Pods: run your first one", Timeout: 120 * time.Second,
		Check: func(e *Env) error {
			img, err := e.get("pod/hello", "{.spec.containers[0].image}")
			if err != nil {
				return err
			}
			if !strings.HasPrefix(img, "nginx:") {
				return fmt.Errorf("pod hello should run an nginx image, has %q", img)
			}
			return e.podRunning("hello")
		},
	},
	{
		ID: "02_env_vars", Title: "Pods: environment variables", Timeout: 90 * time.Second,
		Check: func(e *Env) error {
			if err := e.podRunning("greeter"); err != nil {
				return err
			}
			out, err := e.logs("greeter")
			if err != nil {
				return err
			}
			if !strings.Contains(out, "Hello, k8slings") {
				return fmt.Errorf("pod logs should contain 'Hello, k8slings', got %q", firstLine(out))
			}
			return nil
		},
	},
	{
		ID: "03_labels_selectors", Title: "Deployments: labels and selectors", Timeout: 120 * time.Second,
		Check: func(e *Env) error {
			return e.expect("deploy/web", "{.status.readyReplicas}", "2", "ready replicas of deployment web")
		},
	},
	{
		ID: "04_scale_and_image", Title: "Deployments: scale and change the image", Timeout: 150 * time.Second,
		Check: func(e *Env) error {
			img, err := e.get("deploy/api", "{.spec.template.spec.containers[0].image}")
			if err != nil {
				return err
			}
			if img != "nginx:1.27" {
				return fmt.Errorf("deployment api should use nginx:1.27, has %q", img)
			}
			return e.expect("deploy/api", "{.status.readyReplicas}", "3", "ready replicas of deployment api")
		},
	},
	{
		ID: "05_rolling_update", Title: "Deployments: zero-downtime rollout strategy", Timeout: 120 * time.Second,
		Check: func(e *Env) error {
			if err := e.expect("deploy/shop", "{.spec.strategy.type}", "RollingUpdate", "strategy type"); err != nil {
				return err
			}
			if err := e.expect("deploy/shop", "{.spec.strategy.rollingUpdate.maxUnavailable}", "0", "maxUnavailable"); err != nil {
				return err
			}
			if err := e.expect("deploy/shop", "{.spec.strategy.rollingUpdate.maxSurge}", "1", "maxSurge"); err != nil {
				return err
			}
			return e.expect("deploy/shop", "{.status.readyReplicas}", "3", "ready replicas of deployment shop")
		},
	},
	{
		ID: "06_service_selector", Title: "Services: make traffic reach the pods", Timeout: 120 * time.Second,
		Check: func(e *Env) error {
			out, err := kubectl("get", "endpointslice", "-n", e.NS,
				"-l", "kubernetes.io/service-name=web",
				"-o", `jsonpath={.items[*].endpoints[?(@.conditions.ready==true)].addresses[0]}`)
			if err != nil {
				return fmt.Errorf("cannot read endpoints: %s", firstLine(out))
			}
			if strings.TrimSpace(out) == "" {
				return fmt.Errorf("service web has no ready endpoints; its selector matches no pod")
			}
			ports, _ := kubectl("get", "endpointslice", "-n", e.NS,
				"-l", "kubernetes.io/service-name=web",
				"-o", `jsonpath={.items[*].ports[*].port}`)
			if strings.TrimSpace(ports) != "80" {
				return fmt.Errorf("service web forwards to port %q but nginx listens on 80", strings.TrimSpace(ports))
			}
			return nil
		},
	},
	{
		ID: "07_configmap", Title: "Config: ConfigMap keys", Timeout: 90 * time.Second,
		Check: func(e *Env) error {
			if err := e.podRunning("app"); err != nil {
				return err
			}
			out, err := e.logs("app")
			if err != nil {
				return err
			}
			if !strings.Contains(out, "level=debug") {
				return fmt.Errorf("pod logs should contain 'level=debug', got %q", firstLine(out))
			}
			return nil
		},
	},
	{
		ID: "08_probes", Title: "Probes: why is my pod never Ready?", Timeout: 120 * time.Second,
		Check: func(e *Env) error {
			probe, err := e.get("pod/site", "{.spec.containers[0].readinessProbe}")
			if err != nil {
				return err
			}
			if probe == "" {
				return fmt.Errorf("keep the readinessProbe, fix it instead of deleting it")
			}
			return e.expect("pod/site", `{.status.conditions[?(@.type=="Ready")].status}`, "True", "pod Ready condition")
		},
	},
	{
		ID: "09_resources", Title: "Resources: requests and limits", Timeout: 90 * time.Second,
		Check: func(e *Env) error {
			lim, err := e.get("pod/worker", "{.spec.containers[0].resources.limits.memory}")
			if err != nil {
				return err
			}
			if lim == "" {
				return fmt.Errorf("pod worker needs a memory limit; don't just delete the resources block")
			}
			return e.podRunning("worker")
		},
	},
	{
		ID: "10_job", Title: "Jobs: run to completion", Timeout: 90 * time.Second,
		Check: func(e *Env) error {
			return e.expect("job/hello", "{.status.succeeded}", "1", "succeeded pods of job hello")
		},
	},
	{
		ID: "11_rbac", Title: "RBAC: least privilege for a ServiceAccount", Timeout: 30 * time.Second,
		Check: func(e *Env) error {
			sa := fmt.Sprintf("system:serviceaccount:%s:reader", e.NS)
			can := func(verb string) string {
				out, _ := kubectl("auth", "can-i", verb, "pods", "-n", e.NS, "--as", sa)
				return strings.TrimSpace(out)
			}
			if can("list") != "yes" {
				return fmt.Errorf("serviceaccount reader cannot list pods yet")
			}
			if can("get") != "yes" {
				return fmt.Errorf("serviceaccount reader cannot get pods yet")
			}
			if can("delete") != "no" {
				return fmt.Errorf("serviceaccount reader must NOT be able to delete pods (least privilege)")
			}
			return nil
		},
	},
	{
		ID: "12_secrets", Title: "Secrets: inject a credential", Timeout: 90 * time.Second,
		Check: func(e *Env) error {
			ref, err := e.get("pod/app", "{.spec.containers[0].env[0].valueFrom.secretKeyRef.name}")
			if err != nil {
				return err
			}
			if ref != "db-creds" {
				return fmt.Errorf("pod app should read DB_PASSWORD from Secret db-creds via secretKeyRef, has %q", ref)
			}
			if err := e.podRunning("app"); err != nil {
				return err
			}
			out, err := e.logs("app")
			if err != nil {
				return err
			}
			if !strings.Contains(out, "password-length=11") {
				return fmt.Errorf("pod logs should contain 'password-length=11', got %q", firstLine(out))
			}
			return nil
		},
	},
	{
		ID: "13_pvc_storage", Title: "Storage: a claim that binds", Timeout: 90 * time.Second,
		Check: func(e *Env) error {
			if err := e.expect("pvc/notes", "{.status.phase}", "Bound", "PVC notes phase"); err != nil {
				return err
			}
			if err := e.podRunning("writer"); err != nil {
				return err
			}
			out, err := e.logs("writer")
			if err != nil {
				return err
			}
			if !strings.Contains(out, "saved") {
				return fmt.Errorf("pod logs should contain 'saved', got %q", firstLine(out))
			}
			return nil
		},
	},
	{
		ID: "14_init_container", Title: "Pods: init container and shared volume", Timeout: 90 * time.Second,
		Check: func(e *Env) error {
			if err := e.expect("pod/web", "{.status.initContainerStatuses[0].state.terminated.exitCode}", "0", "init container exit code"); err != nil {
				return err
			}
			if err := e.podRunning("web"); err != nil {
				return err
			}
			out, err := e.exec("web", "nginx", "cat", "/usr/share/nginx/html/index.html")
			if err != nil {
				return fmt.Errorf("cannot read index.html in the nginx container: %s", firstLine(out))
			}
			if !strings.Contains(out, "hello from init") {
				return fmt.Errorf("nginx should serve the file written by the init container, got %q", firstLine(out))
			}
			return nil
		},
	},
	{
		ID: "15_scheduling", Title: "Scheduling: nodeSelector and affinity", Timeout: 90 * time.Second,
		Check: func(e *Env) error {
			sel, err := e.get("deploy/batch", "{.spec.template.spec.nodeSelector}")
			if err != nil {
				return err
			}
			aff, _ := e.get("deploy/batch", "{.spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution}")
			if sel == "" || aff == "" {
				return fmt.Errorf("keep both the nodeSelector and the required nodeAffinity, fix them instead of deleting them")
			}
			return e.expect("deploy/batch", "{.status.readyReplicas}", "2", "ready replicas of deployment batch")
		},
	},
	{
		ID: "16_networkpolicy", Title: "NetworkPolicy: allow only the client", Timeout: 90 * time.Second,
		Check: func(e *Env) error {
			if _, err := e.get("networkpolicy/default-deny", "{.metadata.name}"); err != nil {
				return fmt.Errorf("keep the default-deny policy")
			}
			for _, p := range []string{"server", "client", "intruder"} {
				if err := e.podRunning(p); err != nil {
					return err
				}
			}
			if out, err := e.exec("client", "", "wget", "-qO-", "-T", "3", "http://server"); err != nil {
				return fmt.Errorf("client cannot reach service server yet: %s", firstLine(out))
			}
			if _, err := e.exec("intruder", "", "wget", "-qO-", "-T", "3", "http://server"); err == nil {
				return fmt.Errorf("intruder can reach server; only the client may")
			}
			return nil
		},
	},
	{
		ID: "17_rbac_write", Title: "RBAC: write the policy yourself", Timeout: 30 * time.Second,
		Check: func(e *Env) error {
			for _, p := range []struct{ verb, res, ns, want string }{
				{"get", "pods", e.NS, "yes"},
				{"list", "pods", e.NS, "yes"},
				{"watch", "pods", e.NS, "yes"},
				{"get", "pods/log", e.NS, "yes"},
				{"get", "deployments.apps", e.NS, "yes"},
				{"list", "deployments.apps", e.NS, "yes"},
				{"patch", "deployments.apps", e.NS, "yes"},
				{"create", "deployments.apps", e.NS, "no"},
				{"delete", "deployments.apps", e.NS, "no"},
				{"delete", "pods", e.NS, "no"},
				{"get", "secrets", e.NS, "no"},
				{"list", "pods", "kube-system", "no"},
			} {
				if got := e.canI("deployer", p.verb, p.res, p.ns); got != p.want {
					return fmt.Errorf("deployer: can-i %s %s in %s is %q, want %q", p.verb, p.res, p.ns, got, p.want)
				}
			}
			return nil
		},
	},
	{
		ID: "18_quota_limits", Title: "Quotas: fit the Deployment under the namespace quota", Timeout: 90 * time.Second,
		Check: func(e *Env) error {
			if err := e.expect("resourcequota/team-quota", `{.spec.hard.requests\.cpu}`, "500m", "quota requests.cpu (do not change the quota)"); err != nil {
				return err
			}
			req, err := e.get("deploy/web", "{.spec.template.spec.containers[0].resources.requests.cpu}")
			if err != nil {
				return err
			}
			if req == "" {
				return fmt.Errorf("set an explicit resources.requests.cpu on the container")
			}
			return e.expect("deploy/web", "{.status.readyReplicas}", "3", "ready replicas of deployment web")
		},
	},
	{
		ID: "19_broken_stack", Title: "Capstone: debug the broken stack", Timeout: 150 * time.Second,
		Check: func(e *Env) error {
			if lim, _ := e.get("deploy/site", "{.spec.template.spec.containers[0].resources.limits.memory}"); lim == "" {
				return fmt.Errorf("keep a memory limit on the container, just make it realistic")
			}
			if probe, _ := e.get("deploy/site", "{.spec.template.spec.containers[0].readinessProbe}"); probe == "" {
				return fmt.Errorf("keep the readinessProbe, fix it instead of deleting it")
			}
			if err := e.expect("deploy/site", "{.status.readyReplicas}", "2", "ready replicas of deployment site"); err != nil {
				return err
			}
			if err := e.podRunning("client"); err != nil {
				return err
			}
			out, err := e.exec("client", "", "wget", "-qO-", "-T", "3", "http://site")
			if err != nil {
				return fmt.Errorf("client cannot reach service site yet: %s", firstLine(out))
			}
			if !strings.Contains(out, "k8slings stack ok") {
				return fmt.Errorf("service site should return 'k8slings stack ok', got %q", firstLine(out))
			}
			return nil
		},
	},
}

func findExercise(id string) *Exercise {
	num := strings.TrimLeft(id, "0")
	for _, ex := range registry {
		parts := strings.SplitN(ex.ID, "_", 2)
		if ex.ID == id || strings.TrimLeft(parts[0], "0") == num || parts[1] == id {
			return ex
		}
	}
	return nil
}
