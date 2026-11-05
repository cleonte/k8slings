# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- The first Go version of the CLI, built like golings: `init`, `run`, `hint`, `list`, `verify`, `reset` and `solution`, with the exercises and solutions embedded in the binary.
- Manifest mode: exercises are checked offline, against the embedded Kubernetes 1.37 JSON schemas (unknown fields are errors) and CEL assertions from a `check.yaml` that stays inside the binary.
- 15 exercises: pods, labels and selectors, namespaces, deployments, config (ConfigMap and Secret references), probes and jobs; six of them migrated from the prototype with their runtime checks turned into manifest checks.
- `TestSolutions`: every exercise must fail as shipped and pass with its solution.
- `k8slings watch`: a full-screen view (Bubble Tea v2) that checks the next exercise whenever you save a file. Keys: `h` hint, `s` solution (asks first), `r` reset (asks first), `n` next, `l` list with filter, `c` check all, `f` full output, `q` quit. `--plain` prints lines instead and is used automatically when input or output is not a terminal.

- `k8slings doctor`: two halves. Offline mode: the manifest, that every exercise file is in place, the embedded Kubernetes schema version. Cluster mode: the container runtime (Docker Desktop, Colima, Lima, Linux, ...), kind, kubectl, the Docker host's inotify limits with the fix for your runtime, and the health of the cluster (nodes, CoreDNS, kube-proxy, kindnet, storage, version skew). Missing tools and a missing cluster are warnings, since manifest exercises do not need them; a broken cluster is a failure.
- README demo GIFs, recorded with vhs from `demo.tape` and `demo-tui.tape`.
- A weekly macOS smoke test (`go install`, then the first commands a learner types).

- `k8slings cluster up|down|status`: one kind cluster of its own (one control plane and two workers, `--single` for one node) on a Kubernetes 1.37 node image pinned by digest. It has its own kubeconfig, so `~/.kube/config` and your current context are never touched. `up` checks the Docker host's inotify limit first and prints the fix for Lima, Colima or Linux, waits until every node is Ready and kube-proxy is Running, and retries a failed first start once.
- `k8slings kubectl ...`: kubectl pinned to the k8slings cluster; flags that choose a cluster, user or credentials are refused.
- Before cluster exercises run, the kubeconfig is checked against the control-plane container's published API server port, and anything else is refused.

- Cluster exercises. `check.yaml` can have `steps` that are polled against the live cluster (`rollout`, `ready`, `assert`, `endpoints`, `http`, `exec`, `can_i`, `logs`). A cluster exercise is applied to its own namespace, `k8slings-<name>`, which is emptied on every run; typos in the manifest are reported before anything is applied, and a mistake the cluster cannot fix (`now: true`) ends the run at once. When it does not work in time, the failure shows the problem and the Pods and warnings of the namespace.
- Without a cluster, cluster exercises are skipped with `run \`k8slings cluster up\``, and `watch` and `run next` carry on with the manifest exercises.
- `cluster up` pulls the images the exercises use (nginx, busybox) into every node.
- Two cluster exercises: `deployments2` (rolling update) and `services1` (a Service whose targetPort is wrong).
- `make cluster-test`: creates real clusters and proves the safety rules and every cluster exercise.

### Changed

- The earlier prototype (a cluster-only version with 19 exercises) moved to `prototype/` until its exercises are migrated.
