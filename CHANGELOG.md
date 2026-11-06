# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Setups with several steps** for fix exercises: `setup.yaml`, then `setup-2.yaml` and so on (up to five). Each step except the last has to finish rolling out before the next is applied, so an exercise can start from a cluster with history. Seeding shows "setting up <name>: step 1 of 2" in `run`, `watch --plain` and the full-screen view, and cancelling (`q`, ctrl+c) stops it with nothing recorded, so the next run starts over. A seeding that fails leaves no recorded state either. `init`, `reset` and `doctor` all decide which files stay in the binary with one function, so no setup step is ever written to your folder.
- `troubleshooting12`: a bad rollout. Revision 1 works, revision 2 asks for an image tag that does not exist, and the ticket is "get it back to what was running yesterday" (`kubectl rollout history`, `kubectl rollout undo`). Setting the old image by hand is accepted too: the check cannot tell it from an undo.
- `capstone2`: three unrelated faults in one namespace (a misspelled image, a Service selector that matches no Pod, a ConfigMap key that is not the one the Pod asks for), and a ticket that only describes the symptoms. The check always names the first fault that is left, and every object has to be repaired in place.

### Added

- Five more troubleshooting exercises: `troubleshooting7` (a ResourceQuota that rejects Pods without requests), `troubleshooting8` (a NetworkPolicy that forgets DNS), `troubleshooting9` (OOMKilled), `troubleshooting10` (a PersistentVolumeClaim with a StorageClass that does not exist), and `troubleshooting11` (node affinity with a mistyped node name; needs the default three-node cluster). The proofs refuse the shortcuts: raising or deleting the quota, allowing all egress, removing or inflating the memory limit, swapping the volume for `emptyDir`, removing the affinity, `nodeName`, and relabelling a node.

### Changed

- Closed cheats in earlier exercises: `troubleshooting2` (another image), `troubleshooting4` (removing the CPU request), `troubleshooting5` (a TCP probe instead of HTTP), `troubleshooting6` (binding the `view` ClusterRole; it also checks that `reader` cannot read ConfigMaps, list Deployments or create Pods).

## [0.4.0] - 2026-11-06

### Security

- `k8slings kubectl` now refuses `-s` with its value attached (`-shttps://other:6443`), which slipped past the check for flags that choose the cluster.
- Emptying or seeding a namespace is refused unless its name starts with `k8slings-`, so no bug can wipe `default` or `kube-system`.

### Fixed

- `troubleshooting1` no longer accepts replacing the container command with `sleep 3600`: the application has to run in production mode, and the proof requires that shortcut to be refused.

### Added

- Five more troubleshooting exercises (fix mode): `troubleshooting2` (ImagePullBackOff from a typo in the image name), `troubleshooting3` (a Service whose selector matches no Pod), `troubleshooting4` (a Pod stuck Pending because it asks for more CPU than any node has), `troubleshooting5` (Running but never Ready: a readiness probe on a path that does not exist), `troubleshooting6` (Forbidden: a RoleBinding that names the wrong ServiceAccount). Each one refuses the usual shortcuts (recreate the object, scale to zero, remove the probe, widen the Role, pin the Pod to a node, relabel the Pods by hand), and the proof on a real cluster requires it.

## [0.3.0] - 2026-11-06

### Added

- **Fix mode**, the third kind of exercise: k8slings breaks the cluster on purpose and hands you a ticket, as in a real incident, and you repair it with `kubectl` (`k8slings kubectl ...`). There is no file to edit. The first `run` or `watch` seeds the broken state and says so; after that nothing ever seeds it again by itself, so your repair is never wiped. `watch` looks at the cluster every couple of seconds and turns green on its own. A repair has to hold for a few seconds before it counts (a crash-looping container is briefly Ready), deleting an object and creating a new one with the same name is refused, and `reset` breaks the cluster again. Completion is kept in `.k8slings.json`.
- `troubleshooting1`: a Deployment whose Pods keep restarting (CrashLoopBackOff). Its proof tries three ways to cheat (recreate the Deployment, swap the image, scale it to zero) and requires each to be refused.
- `verify` and check-all never break the cluster for a fix exercise you have not started.

### Quality

- `TestFixSolutions` proves each fix exercise on a real cluster: seed, fail for the declared reason, the solution script repairs it and it holds, every shortcut refused. It shares one cluster with the other proofs, so CI does not pay for a second one.

## [0.2.1] - 2026-11-06

### Changed

- Check all (`c`) in `k8slings watch` no longer freezes the screen. It checks four exercises at a time and shows the state of each as it goes (waiting, running, works, fails, skipped) with a counter, `esc` cancels what is left, a second `c` says it is already checking instead of starting another one, and a save does not throw away a check that is half way. With cluster exercises that matter: a learner who has not done them yet used to wait for every one of them to time out, one after the other, with nothing on screen.
- The list of results also says when an exercise works but still has its `I AM NOT DONE` line, and flags one that was finished and now fails.
- The footer no longer offers `f full`, because nothing is ever shortened and the key showed nothing. The header says "N skipped (see k8slings doctor)" when exercises are skipped, so you know where to look.

### Quality

- Many more tests of the full-screen view, run with the race detector: the limit on how many exercises are checked at once, cancelling, saving during a check, skipped exercises, the filter (where `q` is a letter, not a quit key), scrolling, resizing, and the error, all-done and nothing-to-run screens.

## [0.2.0] - 2026-11-06

### Added

- Nine exercises, which makes 35 in 17 topics: `jobs2` and `jobs3` (a CronJob's cron schedule, overlapping runs and kept history), `statefulsets1` (one volume per replica with `volumeClaimTemplates`), `statefulsets2` (a StatefulSet whose `serviceName` names no Service, so its Pods get no DNS name; on the cluster), `daemonsets1` (a DaemonSet that must tolerate the control-plane taint to run on every node, on the multi-node cluster), and `security1` to `security4` (running as non-root, a read-only root filesystem with a writable scratch volume, least-privilege capabilities, and the "restricted" Pod Security Standard, checked control by control).
- A note in the README on why macOS may block a binary downloaded in a browser, and how to get past it.

### Quality

- Every cluster exercise now says why it fails as shipped (`shipped:` in its `check.yaml`), and the real-cluster proof requires the unsolved exercise to fail with exactly that. Failing alone proves little: a slow cluster fails everything. This caught an exercise (`statefulsets2`) whose supposed fault was not a fault.
- Checks use fully qualified in-cluster names, so a lookup cannot fall back to public DNS.

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

[Unreleased]: https://github.com/cleonte/k8slings/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/cleonte/k8slings/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/cleonte/k8slings/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/cleonte/k8slings/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/cleonte/k8slings/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/cleonte/k8slings/releases/tag/v0.1.0
