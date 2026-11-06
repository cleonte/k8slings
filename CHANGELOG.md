# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Nine exercises, which makes 35 in 17 topics: `jobs2` and `jobs3` (a CronJob's cron schedule, overlapping runs and kept history), `statefulsets1` (one volume per replica with `volumeClaimTemplates`), `statefulsets2` (a stable DNS name per Pod through a headless Service, on the cluster), `daemonsets1` (a DaemonSet that must tolerate the control-plane taint to run on every node, on the multi-node cluster), and `security1` to `security4` (running as non-root, a read-only root filesystem with a writable scratch volume, least-privilege capabilities, and the "restricted" Pod Security Standard, checked control by control).
- A note in the README on why macOS may block a binary downloaded in a browser, and how to get past it.

## [0.1.0] - 2026-11-06

The first release: a Rustlings-style course for Kubernetes, built like [golings](https://github.com/cleonte/golings). 26 exercises in 14 topics, in two modes.

### Learning

- `k8slings init` writes the exercises into a folder; `k8slings watch` is a full-screen view (Bubble Tea v2) that checks the next exercise every time you save a file. Keys: `h` hint, `s` solution (asks first), `r` reset (asks first), `n` next, `l` list with filter, `c` check all, `f` full output, `q` quit. `--plain` prints lines instead, and is used automatically when input or output is not a terminal.
- Also `run`, `hint`, `list`, `verify`, `reset` and `solution`, as in golings. Each exercise has a lesson (`README.md`), a `# TODO:` at the fix site, a hint and a link to the Kubernetes docs. Delete the `# I AM NOT DONE` line when you are done.
- `k8slings doctor` checks the workspace and what it needs, in two halves, and says whether offline mode and cluster mode are ready.

### Manifest exercises (offline, no cluster)

- 16 of the exercises run with nothing but the binary. Your YAML is validated against the embedded Kubernetes 1.37 JSON schemas (unknown fields are errors, so a misspelled `contianers:` is caught) and checked with CEL assertions. The assertions live inside the binary, so they are not a spoiler.
- `quantity()` in CEL compares resources such as `64Gi` with `64Mi`, or `500m` with `1`.

### Cluster exercises

- 10 exercises run against a local [kind](https://kind.sigs.k8s.io/) cluster of their own: rollouts, a Service that finds Pods but the wrong port, an init container, a claim that never binds, scheduling constraints, RBAC as the API server sees it (including pod logs), a namespace quota, network policies that must let one Pod in and keep another out, and a broken stack with four faults.
- `k8slings cluster up|down|status` creates one control plane and two workers (`--single` for one node) on a Kubernetes 1.37 node image pinned by digest, and pulls the images the exercises use into every node. `up` checks the Docker host's inotify limit first and prints the fix for your runtime (Lima, Colima, Linux), waits until every node is Ready and kube-proxy is Running, and retries a failed first start once.
- Each exercise is applied to its own namespace, `k8slings-<name>`, which is emptied on every run. The checks are read-only and polled: rollouts, readiness, endpoints, requests through the API server's service proxy, commands in Pods, `kubectl auth can-i`, logs. A failure shows the problem, and the Pods and warnings of the namespace.
- Without a cluster, these exercises are skipped with the command to run (`k8slings cluster up`), and `watch` and `run next` carry on with the manifest exercises.

### Safety

- k8slings only ever talks to its own cluster. It has its own kubeconfig file, so `~/.kube/config` and your current context are never touched. Every kubectl call names that file and the context and runs without `$KUBECONFIG`; flags that choose a cluster, user or credentials (`--context`, `--server`, ...) are refused; and before cluster exercises run, the kubeconfig is checked against the API server port that Docker publishes for the control-plane container.
- `k8slings kubectl ...` is kubectl pinned to that cluster.

### Quality

- `TestSolutions`: every exercise must fail as shipped and pass with its solution. Cluster exercises are proven the same way on a real kind cluster (`make cluster-test`), which also runs in CI on every pull request that touches them.
- CI runs on Linux (Go 1.26, 1.27 and stable) and Windows, a weekly macOS smoke test, and the cluster job.
- A weekly job opens an issue when a newer kind release or Kubernetes node image than the pinned ones is out.

[Unreleased]: https://github.com/cleonte/k8slings/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/cleonte/k8slings/releases/tag/v0.1.0
