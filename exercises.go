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
}

func findExercise(id string) *Exercise {
	for _, ex := range registry {
		if ex.ID == id || strings.TrimLeft(strings.SplitN(ex.ID, "_", 2)[0], "0") == id ||
			strings.SplitN(ex.ID, "_", 2)[1] == id {
			return ex
		}
	}
	return nil
}
