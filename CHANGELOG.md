# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Query mode**, a fourth kind of exercise: you write one `kubectl` command in `answer.kubectl`, and k8slings runs it on its own cluster and compares what it prints with what was asked for. The text is never run in a shell: a tokenizer reads the safe part of POSIX quoting, only read-only verbs (`get`, `describe`, `explain`, `logs`, `events`, `auth can-i`, `api-resources`, and `create ... --dry-run=client`) and the flags listed for each are accepted, and `-n` is the exercise's own namespace. The output is compared with the reference answer's on the same cluster state, and then once more with more objects in the namespace, so typing in today's names does not pass. Some exercises check the printed YAML as a manifest instead; those need no cluster, only `kubectl`.
- The `kubectl` topic, 13 query exercises in three places of the course: label and set-based selectors, `--field-selector`, `-o jsonpath` with `range` and filters, `custom-columns`, `--sort-by`, generating a Deployment and a Service, `kubectl explain`, `auth can-i --list --as`, and `logs --previous`.

## [0.12.1] - 2026-11-07

### Changed

- Four exercises of 0.12.0 repeated an earlier lesson and were **replaced**, keeping their names and places: `daemonsets4` (now a node agent with `hostNetwork` and a read-only `hostPath`; the control-plane toleration is `daemonsets2`), `statefulsets5` (now `persistentVolumeClaimRetentionPolicy`; the governing Service is `statefulsets2`), `serviceaccounts3` (now `imagePullSecrets` on a ServiceAccount; the token is `serviceaccounts1`) and `lifecycle4` (now two init containers that must run in order; shared volumes are `pods6`). `k8slings update` brings the new versions into a folder where you have not edited the old ones, and lists the others as newer. If you had already finished one of the old versions, it counts as done.
- `troubleshooting20` needs a toleration that names the control-plane taint: a toleration with no key, which tolerates every taint, is refused.
- `networkpolicies5` needs DNS on TCP port 53 as well as UDP.
- `troubleshooting14` says that deleting the Pods is an accepted repair, as is `rollout restart`; `troubleshooting19` says that the logs show the port (the debug container is for applications that print nothing).

### Fixed

- Resetting an exercise namespace no longer removes the finalizers of the cluster's own controllers when they are on a subdomain, such as `batch.kubernetes.io/job-tracking` on the Pods of a Job. Every `kubernetes.io` and `k8s.io` domain, subdomains included, is left alone.

### Added

- A test that fails when two exercises of a topic have the same README title.

## [0.12.0] - 2026-11-07

### Added

- 13 offline exercises for the topics that had the fewest: `disruption3` (a budget whose selector matches nothing), `disruption4` (`minAvailable` and `maxUnavailable` together), `lifecycle4` (a volume only the init container mounts), `daemonsets4` (the control-plane taint), `networkpolicies4` (`from` entries that are alternatives), `networkpolicies5` (egress that forgets DNS), `resources4` (a quota that counts requests), `serviceaccounts3` (no token mounted), `security5` (`privileged` instead of one capability), `rbac5` (`pods/log`), `probes4` (a named port that does not exist), `statefulsets5` (the headless governing Service) and `namespaces4` (a quota in the wrong namespace). A folder from an older version gets them with `k8slings update`.

### Changed

- An exercise is done only when its check has passed **and** its `# I AM NOT DONE` line is gone. Before, deleting the line was enough, so a file that still failed could be skipped by accident (`run` now says so when that is the case). Passes are recorded in `.k8slings.json`, as fix exercises already did; `reset` forgets the pass. A folder made by an older version keeps every exercise whose line was already gone: they are credited once, the first time a command reads it.
- The cluster proofs in CI are split by a hash of the exercise name instead of by position, and the safety rules run in a job of their own.

## [0.11.0] - 2026-11-07

### Added

- `k8slings update` adds the exercises of a newer version of k8slings to an exercises folder that an older version made. Before, new exercises never reached an existing folder, because `init` copies them once. The new ones go where they belong in the order of the course (`list` marks them `new`), `info.toml` is kept as `info.toml.bak`, your work is never overwritten, READMEs and exercise files you have not changed are replaced by their newer versions, and `--dry-run` shows what would happen. `run`, `list`, `watch` and `doctor` tell you when your folder is behind. For a folder made by an older version (before this one) it can add exercises, but not tell which files you changed, so it leaves them alone.

## [0.10.0] - 2026-11-07

### Added

- Three more troubleshooting exercises: `troubleshooting18` (a Job that gave up: it has to be made again with the right path), `troubleshooting19` (an application whose image has no shell: `kubectl debug` with an ephemeral container) and `troubleshooting20` (a Pod that has to run on the control-plane node and does not tolerate its taint; it needs the cluster with its workers). Run `k8slings cluster up` again to preload the image of `troubleshooting19` (`hashicorp/http-echo:1.0.0`); otherwise that exercise pulls it itself, a little slower.
- Four more troubleshooting exercises: `troubleshooting14` (the flag was changed in the ConfigMap, and no Pod sees it: environment variables are read once, `rollout restart`), `troubleshooting15` (a delete that never finishes: a finalizer nobody handles), `troubleshooting16` (a slow starter killed by its liveness probe) and `troubleshooting17` (a CronJob that never runs because it is suspended).

### Changed

- `lifecycle2` and `scheduling4` refuse their cheats at once, before anything is applied. `scheduling4` also refuses a *preferred* anti-affinity or a spread that only schedules anyway: they only work when the scheduler happens to agree.
- Resetting an exercise (and so starting one) works when an object in its namespace carries a finalizer that no controller handles; before, the reset hung for a minute and failed.
- Contributors: a cluster exercise can ship `shortcut-*.yaml` (a cheat as an exercise file, refused at once, proven by `TestClusterSolutions`); `make cluster-test` has a longer timeout, and the CI cluster job runs as five shards (the full run takes about 40 minutes with twenty fix exercises).

## [0.9.0] - 2026-11-07

### Added

- Five scheduling and disruption exercises, and a new topic, `disruption`: `scheduling3` (a toleration that does not match its taint), `scheduling4` (replicas pulled together by `podAffinity` where `podAntiAffinity` was meant; it runs on the cluster and needs the two workers), `scheduling5` (a PriorityClass above the allowed maximum and a Pod that names another one), `disruption1` (a PodDisruptionBudget that forbids every disruption) and `disruption2` (a budget whose status says no eviction is ever allowed; it runs on the cluster).
- Ten workload exercises, and a new topic, `lifecycle`: `lifecycle1` (a pre-stop delay longer than the grace period), `lifecycle2` (a Job that never ends because of its log shipper; the fix is a native sidecar; it runs on the cluster), `lifecycle3` (an application that starts before the helper it needs; native sidecar with a startup probe; it runs on the cluster), `jobs4` (`completions` and `parallelism`), `jobs5` (an Indexed Job), `jobs6` (`activeDeadlineSeconds` and `ttlSecondsAfterFinished`), `jobs7` (`podFailurePolicy`), `deployments5` (a rollout that waits for itself on a ReadWriteOncePod claim), `deployments6` (`revisionHistoryLimit` and `minReadySeconds`) and `daemonsets3` (a DaemonSet update that can take the agent off every node).
- Eleven basics exercises, and a new topic, `serviceaccounts`: `config5` (a Secret with plain text where base64 belongs), `config6` (a TLS Secret with the wrong key names), `pods7` (an unpinned image with `IfNotPresent`), `pods8` (the downward API), `services4` (a `targetPort` name the container does not declare), `services5` (an ExternalName that is a URL), `volumes3` (three volumes on one path, merged into a projected volume), `volumes4` (a ReadWriteMany claim that the default StorageClass cannot provide; it runs on the cluster), `labels5` (`matchExpressions` with an operator that needs values), `serviceaccounts1` (a token that is mounted although the ServiceAccount turns it off) and `serviceaccounts2` (a projected token below the minimum lifetime).

### Changed

- CI: the cluster job runs as four shards instead of three. Contributors only.

## [0.8.0] - 2026-11-07

### Added

- `e` in the full-screen `watch` opens the exercise in your editor (`$VISUAL`, or else `$EDITOR`) and waits until you close it; for a fix exercise it opens the ticket. Without an editor set, the key says how to set one. Same key as in golings.

### Changed

- When an exercise is not solved, the path to edit now ends with the line of its `# TODO:` comment (`exercises/pods/pods1/exercise.yaml:8`), in `run`, `verify` and the full-screen view. Terminals of editors such as VS Code turn it into a link that opens the file at that line.
- `k8slings init` now suggests `k8slings watch` as the next step, as the README does, instead of `run next`.

## [0.7.0] - 2026-11-07

### Added

- Ten more offline exercises: `config3` (an `env` entry hides the ConfigMap), `config4` (a ConfigMap mounted over a whole directory; `subPath`), `resources3` (make a Pod Guaranteed), `statefulsets3` (a StatefulSet whose headless Service does not exist), `statefulsets4` (a canary with `partition`), `networkpolicies3` (a default deny with an empty rule), `namespaces3` (a LimitRange whose default request is above its default limit), `rbac4` (a ServiceAccount subject without its namespace), `deployments4` (a progress deadline shorter than `minReadySeconds`) and `probes3` (a liveness probe that depends on the database).
- `troubleshooting13`: the client's Pods use a DNS server that does not exist (`dnsPolicy: None`), so no name resolves although the Service is fine. A hosts entry for the Service does not count.
- Check steps: `exec` can pick the Pod with a `selector` (the first Pod that is Running and not being deleted), for the Pods of a Deployment, whose names nobody can write down. Contributors only.

### Fixed

- A bare number for a quantity in an exercise (`cpu: 1`, `cpu: 0.5`) is accepted by the checks, as Kubernetes accepts it. Before, a correct answer written that way was refused.

## [0.6.0] - 2026-11-07

### Added

- Ten more offline exercises, one for each of ten topics that had only one or two: `probes2` (a startup probe for a slow application), `services2` (a `targetPort` that nothing listens on), `services3` (a node port outside the allowed range), `volumes2` (a mount of a volume the Pod does not have), `daemonsets2` (the control-plane taint key), `scheduling2` (a spread constraint that selects no Pod of its own Deployment), `networkpolicies2` (two `from` entries are OR, one entry with both selectors is AND), `namespaces2` (a Pod in another namespace than its ConfigMap), `deployments3` (a rolling update with `maxSurge` and `maxUnavailable` both zero) and `rbac3` (a Role for the wrong API group). They check what you wrote, so they need no cluster.

## [0.5.0] - 2026-11-07

### Added

- **Setups with several steps** for fix exercises: `setup.yaml`, then `setup-2.yaml` and so on (up to five). Each step except the last has to finish rolling out before the next is applied, so an exercise can start from a cluster with history. Seeding shows "setting up <name>: step 1 of 2" in `run`, `watch --plain` and the full-screen view, and cancelling (`q`, ctrl+c) stops it with nothing recorded, so the next run starts over. A seeding that fails leaves no recorded state either. `init`, `reset` and `doctor` all decide which files stay in the binary with one function, so no setup step is ever written to your folder.
- `troubleshooting12`: a bad rollout. Revision 1 works, revision 2 asks for an image tag that does not exist, and the ticket is "get it back to what was running yesterday" (`kubectl rollout history`, `kubectl rollout undo`). Setting the old image by hand is accepted too: the check cannot tell it from an undo.
- `capstone2`: three unrelated faults in one namespace (a misspelled image, a Service selector that matches no Pod, a ConfigMap key that is not the one the Pod asks for), and a ticket that only describes the symptoms. The check always names the first fault that is left, and every object has to be repaired in place.
- Five more troubleshooting exercises: `troubleshooting7` (a ResourceQuota that rejects Pods without requests), `troubleshooting8` (a NetworkPolicy that forgets DNS), `troubleshooting9` (OOMKilled), `troubleshooting10` (a PersistentVolumeClaim with a StorageClass that does not exist), and `troubleshooting11` (node affinity with a mistyped node name; needs the default three-node cluster). The proofs refuse the shortcuts: raising or deleting the quota, allowing all egress, removing or inflating the memory limit, swapping the volume for `emptyDir`, removing the affinity, `nodeName`, and relabelling a node.
- Fix-mode docs in the README: what to do when you break the cluster for real (`reset`, `cluster down && cluster up`, what is kept).

### Changed

- Closed cheats in earlier exercises: `troubleshooting2` (another image), `troubleshooting4` (removing the CPU request), `troubleshooting5` (a TCP probe instead of HTTP), `troubleshooting6` (binding the `view` ClusterRole; it also checks that `reader` cannot read ConfigMaps, list Deployments or create Pods).
- The cluster proofs also check the *cause* where the first problem is generic: an optional `shipped_details` in `check.yaml` must show in the diagnosis (`troubleshooting9`: OOMKilled, `10`: the missing StorageClass, `11`: the node affinity that matches nothing). Contributors only; learners see no change.
- CI: the kind job runs as three shards, each on its own cluster (it was cancelled at the 30 minute limit with twelve troubleshooting exercises and a capstone on one cluster). `K8SLINGS_SHARD=i/n` does the split; `make cluster-test` without it still runs everything.

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

[Unreleased]: https://github.com/cleonte/k8slings/compare/v0.12.1...HEAD
[0.12.1]: https://github.com/cleonte/k8slings/compare/v0.12.0...v0.12.1
[0.12.0]: https://github.com/cleonte/k8slings/compare/v0.11.0...v0.12.0
[0.11.0]: https://github.com/cleonte/k8slings/compare/v0.10.0...v0.11.0
[0.10.0]: https://github.com/cleonte/k8slings/compare/v0.9.0...v0.10.0
[0.9.0]: https://github.com/cleonte/k8slings/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/cleonte/k8slings/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/cleonte/k8slings/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/cleonte/k8slings/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/cleonte/k8slings/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/cleonte/k8slings/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/cleonte/k8slings/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/cleonte/k8slings/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/cleonte/k8slings/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/cleonte/k8slings/releases/tag/v0.1.0
