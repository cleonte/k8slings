# ROADMAP — k8slings

k8slings is a Rustlings-style course for Kubernetes, built the way [golings](https://github.com/cleonte/golings) is built. This file is the plan; `CLAUDE.md` is the architecture. Both are **design documents awaiting review**: no code changes until they are approved. Phase 0 is where the open decisions are made; the checkboxes in Phase 0 are mostly "decision recorded" rather than "code written".

Counts in this file are planning numbers. Once the consistency test exists (Phase 1), real counts live only between `<!-- count:... -->` markers and the test keeps them equal to `info.toml`.

## Where things stand

Phases 0 to 3 are done: manifest mode, cluster mode (an isolated kind cluster, declarative check steps, a runner with diagnostics, image preload), `watch`, `doctor`, CI on Linux, Windows and a real kind cluster, and the exercises of 14 topics. The first prototype (a cluster-only version with 19 exercises) has been fully migrated and removed from the tree. What is left: fix mode and troubleshooting (Phase 4), add-ons and their topics (Phase 5), and the release (Phase 6).

## Facts checked while writing this design (2026-11-04)

These were looked up, not assumed. Anything not listed here is either from memory or still to be verified in Phase 0.

- Latest kind is **v0.33.0** (2026-08-26). Its default node image is `kindest/node:v1.37.0@sha256:a1ed56cfb0e7b93589bdf97c8cd566405a265939e3620fc4f5de89adff580ae5`; it also lists v1.36.4, v1.35.8 and v1.34.11. Kubernetes v1.37.2 is the newest stable patch; v1.38 is alpha. **Target: Kubernetes 1.37, node image pinned by digest.**
- kind's default CNI, kindnet, has had out-of-the-box NetworkPolicy support (via `sigs.k8s.io/kube-network-policies`) since v0.24, per kind's release notes. So **NetworkPolicy exercises need no add-on**. The prototype's exercise 16 tests real blocked traffic, so it is the proof; it still has to be re-run on the pinned image (Phase 0 task).
- **ingress-nginx is retired**: the Kubernetes project announced it in November 2025 and ended maintenance in March 2026. It is excluded. The Kubernetes community points to Gateway API as the successor; Traefik and Envoy Gateway are the candidates (decided: Traefik, pending spike).
- `yannh/kubernetes-json-schema` (the schema set kubeconform uses) has `v1.37.1-standalone-strict` and `v1.37.2-standalone-strict`. A `v1.37.0` directory was not checked; if absent, use 1.37.1 or generate from `kubernetes/kubernetes` `api/openapi-spec`. Either way the schemas are embedded, never downloaded at runtime.
- Local toolchain: Go 1.27.3, kind v0.33.0, kubectl v1.37.2, so a cluster and a client at the same minor version.
- golings is Apache-2.0 (`LICENSE`, `NOTICE` crediting mauricioabreu/golings); the source material in `../learn/k8s-2-weeks` is MIT (code) and CC BY 4.0 (text).
- Not yet verified: latest Gateway API, metrics-server and the chosen controller's versions and whether the controller runs acceptably in a 3-node kind cluster in a 6 GB Colima VM (Phase 0 and 5 tasks).

## Design summary (what Phase 0 asks you to approve)

**Three modes**, in order of preference for each exercise: *manifest* (offline: strict-schema validation plus CEL assertions in an embedded `check.yaml`), *cluster* (apply to a per-exercise namespace in the k8slings kind cluster; declarative steps for waits and live assertions, same CEL), *fix* (the tool seeds a broken live state; the learner repairs it with `kubectl`; the check asserts the end state; solution is a kubectl script). Details and trade-offs are in CLAUDE.md. The shape follows from three constraints: a beginner must be able to practise without Docker; cluster checks must be fast and read-only; the CI proof that "unsolved fails, solution passes" must work for every exercise in every mode.

**Cluster safety:** own kubeconfig file, never `~/.kube/config`, `KUBECONFIG` ignored, context `kind-k8slings` only, API-server/container cross-check, no override flag.

**Topology:** 1 control plane + 2 workers by default (`--single` for small machines), decided at create time because it cannot be changed later.

### Topics

Order is learning order, which is not the order in the brief: namespaces move forward (every later topic leans on them), resources come before probes' runtime effects, and Kustomize sits after ConfigMaps because it is about generating them. Counts are planning numbers.

| # | Topic | Approx. exercises | Modes | Needs |
|---|---|---|---|---|
| 1 | Pods | 5 | manifest | |
| 2 | Labels and selectors | 4 | manifest | |
| 3 | Namespaces | 3 | manifest | |
| 4 | Deployments and rollouts | 6 | 4 manifest, 2 cluster | |
| 5 | Services | 5 | 3 manifest, 2 cluster | |
| 6 | ConfigMaps and Secrets | 5 | 4 manifest, 1 cluster | |
| 7 | Kustomize | 4 | manifest | `kustomize` build (decided: embed the library) |
| 8 | Probes | 4 | 3 manifest, 1 cluster | |
| 9 | Resources, requests and limits (incl. quota and LimitRange) | 5 | 3 manifest, 2 cluster | |
| 10 | Jobs and CronJobs | 4 | 3 manifest, 1 cluster | |
| 11 | Volumes and PVCs | 5 | 2 manifest, 3 cluster | |
| 12 | StatefulSets | 4 | 2 manifest, 2 cluster | |
| 13 | DaemonSets | 3 | cluster | `multinode` |
| 14 | Scheduling: taints, tolerations, affinity | 5 | 2 manifest, 3 cluster | `multinode` |
| 15 | Security contexts and Pod Security | 5 | 4 manifest, 1 cluster | |
| 16 | RBAC | 5 | 3 manifest, 2 cluster | |
| 17 | Network policies | 4 | 1 manifest, 3 cluster | |
| 18 | HPA | 3 | cluster | `metrics-server` |
| 19 | Ingress and the Gateway API | 5 | 3 manifest, 2 cluster | `gateway` |
| 20 | Troubleshooting: CrashLoopBackOff, ImagePullBackOff, Pending, broken selectors, bad probes, quota, RBAC denials | 8–10 | fix | some `multinode` |

That is about 90 exercises in the long run; **v1.0 does not need all of them** (see Decided, item 3). The brief puts Ingress/Gateway and HPA among the later topics; this table keeps them last but before the capstone-style troubleshooting, because troubleshooting exercises can then use any earlier topic as the broken component.

### Where the 19 prototype exercises went

Every exercise of the first prototype has been migrated, with its check re-reviewed (never weakened). The prototype itself is in the git history.

| Prototype | Now | Notes |
|---|---|---|
| 01 pod basics | pods1 (manifest) | |
| 02 env vars | pods3 (manifest) | asserts the env instead of reading logs, which removes a race |
| 03 labels/selectors | labels1 (manifest) | the API server's rejection is reproduced with CEL |
| 04 scale and image | deployments1 (manifest) | |
| 05 rolling update | deployments2 (cluster) | |
| 06 service selector | labels2 (manifest) and services1 (cluster) | split: the selector half and the targetPort half |
| 07 configmap | config1 (manifest) | |
| 08 probes | probes1 (manifest) | |
| 09 resources | resources1 (manifest) | needed `quantity()` in CEL |
| 10 job | jobs1 (manifest) | |
| 11 rbac (read) | rbac1 (cluster) | |
| 12 secrets | config2 (manifest) | |
| 13 pvc storage | volumes1 (cluster) | |
| 14 init container | pods6 (cluster) | |
| 15 scheduling | scheduling1 (cluster, multinode) | redesigned: the old answer targeted the tainted control plane |
| 16 networkpolicy | networkpolicies1 (cluster) | |
| 17 rbac (write) | rbac2 (cluster) | the old check never verified `pods/log` |
| 18 quota/limits | resources2 (cluster) | |
| 19 broken stack | capstone1 (cluster) | |

## Phase 0 — Decisions (this document)

- [ ] Review and approve CLAUDE.md and ROADMAP.md.
- [x] Decisions recorded under "Decided"; no open questions remain.
- [ ] **Modes** recorded: manifest, cluster, fix (CLAUDE.md "Exercise modes").
- [ ] **Assertion format** recorded: `check.yaml` with CEL; cluster steps vocabulary; named-Go-check escape hatch.
- [ ] **Cluster handling** recorded: owned kubeconfig, context pinning, container cross-check, topology, runtime support matrix.
- [ ] **Versions** recorded: Kubernetes 1.37, kind ≥ v0.33.0, node image digest, Go floor 1.26.
- [x] **License** decided: Apache-2.0 (see Decided).
- [x] Spike (throwaway branch, results written here): evaluate 3 representative CEL assertions with `cel-go` against a parsed Deployment, including `all`, `has` and a cross-object `exists`; measure the binary-size cost.
- [x] Spike: strict schema validation of the same Deployment with `santhosh-tekuri/jsonschema` versus kubeconform as a library; confirm `contianers:` is rejected and error messages are learner-friendly; measure embedded schema size for the kinds in the topic list. Record which is chosen.
- [x] Spike: bring up the 3-node cluster with the pinned image and its own kubeconfig on Colima; measure idle RAM and time to ready; run the prototype's exercise 16 to confirm NetworkPolicy enforcement.
- [x] Spike: Kustomize in-process (decided: embed the library): binary-size and dependency cost.

## Phase 0 spike results (2026-11-04)

Run on Apple silicon, Lima `docker` VM (4 CPU, 8 GiB), kind v0.33.0, node image `v1.37.0` pinned by digest. The spike code was throwaway and is not in the repository.

- **Cluster and isolation:** `kind create cluster --kubeconfig <own file>` leaves `~/.kube/config` byte-identical and the current context unchanged (`kind-lab` before and after). Warm create of 1 control plane + 2 workers takes **27 s**; idle RAM about **750 MB** (control plane ~520 MB, workers ~110 MB each), 13 system pods Running. The first create attempt (cold image pull) failed after 263 s; its error was not captured, so cold-start behaviour needs a retry/timeout policy and a captured error in `cluster up`.
- **inotify is a real, current problem on Lima, not just Colima.** The Lima Docker VM ships `fs.inotify.max_user_instances = 128`. With that, `kind create` reported success, but only the control plane registered, workers never joined and kube-proxy crashed with `fsnotify watcher init: too many open files`. After `sysctl -w fs.inotify.max_user_instances=512 fs.inotify.max_user_watches=524288` the same command produced a healthy 3-node cluster. Design consequence: three nodes use more inotify instances than one, so `doctor` must preflight the limit **before** `cluster up`, and `cluster up` must verify every node is Ready and kube-proxy Running rather than trusting kind's exit code. (CLAUDE.md updated.) The sysctl is not persistent across VM restarts; the fix must be printed with a persistent variant.
- **NetworkPolicy confirmed.** With the exercise 16 solution, `client` reaches `server` and `intruder` times out; with the policies deleted, `intruder` reaches `server`. So kindnet on the pinned image enforces NetworkPolicy, and no add-on is needed.
- **CEL works** (`cel.dev/cel-go` v0.33.2; the module path moved from `github.com/google/cel-go`). All four assertion shapes from CLAUDE.md evaluate correctly on a parsed Deployment and Service: `==`, selector-subset-of-labels with `all`, `has()`, and the cross-object `exists`; a failing assertion evaluates to `false` without error.
- **Strict schema validation works** with `santhosh-tekuri/jsonschema/v6` against yannh `v1.37.0-standalone-strict` (which exists). A valid Deployment passes (34 ms including compile); `contianers:` is rejected with `additional properties 'contianers' not allowed` and `missing property 'containers'`. The error text starts with a file-URL line the learner should not see, so the failure shortener must strip it. kubeconform-as-a-library was **not** benchmarked; jsonschema met every requirement, so the decision is jsonschema unless a Phase 1 problem appears.
- **Schema size:** 26 kinds covering the topic list (workloads, Services, config, storage, RBAC, networking, HPA, quota, PDB) are 5.8 MB raw and **0.65 MB gzipped**. Decision: embed gzipped, decompress on demand. Gateway API and Traefik CRDs are separate and handled in Phase 5.
- **Binary size:** cel-go + jsonschema + yaml: **12.8 MB** stripped. kustomize api alone (51 modules): **15.8 MB**. These overlap in shared dependencies; expect roughly 20-25 MB with the TUI. Decision stays: **embed the kustomize library**; revisit only if the real binary exceeds about 30 MB.
- **Traefik:** documentation shows separate Kubernetes Ingress and Gateway API providers that can be enabled together, and a Gateway API conformance report exists (version claims in Traefik's own docs disagree, from v1.1 to v1.5). Running it in kind is still a Phase 5 spike.

## Phase 1 — Skeleton from golings, offline mode, first 10 exercises

Branch `phase-1-skeleton`. Reuse golings code where it fits, keep Apache-2.0 headers and NOTICE, adapt rather than reinvent.

- [x] `go.mod` at Go 1.26, module `github.com/cleonte/k8slings`; package layout from CLAUDE.md; cobra root with `init`, `run`, `hint`, `list`, `verify`, `reset`, `solution` (plain output first).
- [x] `embed.go` with the two FS values (exercises, solutions); `info.toml` loader with validation (every path exists, names unique, modes valid, `requires` known).
- [x] Marker handling (`# I AM NOT DONE`, `# TODO:`), `n`-style mark-done helper, workspace `init`/`reset` that never overwrites.
- [x] Schema package with generated, embedded schemas (`scripts/gen-schemas.sh`), strict mode, learner-friendly error shortening.
- [x] `check` package: `check.yaml` model, CEL evaluator, failure messages; unit tests for both.
- [x] Manifest-mode runner.
- [x] `TestSolutions` for manifest mode, plus manifest tests (every exercise has a solution and vice versa, hints and docs links present, no `metadata.namespace`, every kind used has an embedded schema).
- [x] First 10 exercises: pods (5), labels and selectors (4), namespaces (1), each with README, hint, docs link, TODO, solution.
- [x] Migrate the prototype's manifest-destined exercises once each check has been reviewed: 03→labels1, 04→deployments1, 07→config1, 08→probes1, 10→jobs1, 12→config2. Prototype 01 and 02 are covered by pods1 and pods3. **09 (resources) is not migrated**: its check compares quantities (`64Gi` against `64Mi`), which plain CEL cannot do. Needs a quantity helper in the `check` package (or the Kubernetes CEL quantity library) first; tracked in Phase 3.
- [x] Docs consistency test with `<!-- count:... -->` markers in README and ROADMAP.
- [x] README, CHANGELOG (Keep a Changelog), CONTRIBUTING (adding an exercise, running tests, releasing), `.gitignore`, issue templates (bug, exercise problem, feature), `dependabot.yml` (grouped Go and Actions updates). `LICENSE` and `NOTICE` per the Phase 0 decision.
- [x] CI: build/vet/test on Linux for Go 1.26, 1.27, stable; Windows job; formatting and vet.

## Phase 2 — TUI and tooling parity with golings

Branch `phase-2-tui`.

- [x] Watcher package (debounced, sees atomic saves), copied and adapted.
- [x] Bubble Tea v2 full-screen `watch`: progress, topic, result pane, re-run on save; keys `h s r n l c f q`; scrolling; `--plain` and automatic fallback when not a TTY. (`f` only says nothing was shortened until cluster output exists.)
- [x] `l` list with filter by name or topic; `c` check-all (sequential: offline checks take milliseconds; parallel arrives with cluster mode); `f` full output (placeholder, see above).
- [x] `doctor` (offline part: manifest, exercise files, embedded schema version, exercises that need a cluster). The Docker, kind, kubectl, inotify and cluster checks are in Phase 3. No `--verbose` flags: nothing is shortened yet.
- [x] `solution` gating, `reset` confirmation, failure shortening for schema and CEL errors (the last was done in Phase 1).
- [x] `make smoke` and `scripts/smoke.sh`; macOS weekly smoke workflow; demo tapes for the README (vhs).

## Phase 3 — Cluster mode and its topics

Branch `phase-3-cluster`.

- [x] `cluster` package: kind wrapper with pinned image, owned kubeconfig, context pinning, the single kubectl builder, container cross-check, refusal rules, and a test that no other code path execs kubectl (`TestOnlyExecGoRunsPrograms`: only `cluster/exec.go` may import `os/exec`).
- [x] `cluster up | down | status`, `kubectl` pass-through, `--single`; `up` preflights the inotify limit, verifies every node Ready and kube-proxy Running, and retries a failed cold create once. Proven against a real kind cluster on Lima (`make cluster-test`).
- [x] Image preload during `cluster up` (nginx, busybox) so no exercise depends on a registry pull. Pulls inside each node with `crictl`: `kind load docker-image` failed on this Docker (containerd image store).
- [ ] Add-on registry skeleton (`requires` handling, skip messages, `list` locks); `multinode` topology.
- [x] `doctor` cluster part: runtime detection (Docker Desktop, Colima, Lima, Rancher Desktop, Podman, Linux), kind/kubectl versions and skew, inotify limits with the fix for the runtime, node and core-pod health. Verified on a real cluster, including a stopped worker.
- [x] Declarative cluster `check.yaml` steps (`rollout`, `ready`, `assert`, `endpoints`, `http` via API-server proxy, `exec`, `can_i`, `logs`). The named-Go-check escape hatch is not built: no exercise needs it yet.
- [x] Cluster-mode runner (namespace reset, apply, poll, diagnostics, per-exercise timeouts, `now` assertions that fail at once). A global ceiling is not built.
- [x] `TestClusterSolutions` behind `-tags cluster`; `make cluster-test` (unsolved fails by the exercise's own check, solution passes, on a real cluster).
- [x] Migrate the prototype's cluster-destined exercises and remove `prototype/`. All 19 are migrated; see the table above.
- [ ] Topics: deployments and rollouts, services, probes (runtime), resources and quota, jobs and CronJobs, volumes and PVCs, StatefulSets, DaemonSets, scheduling.
- [ ] Remaining offline topics from the table (ConfigMaps/Secrets, Kustomize, security contexts, RBAC, network policies shape).
- [x] Cluster CI job on every PR that touches exercises, solutions or the cluster code (path-filtered), on `main` and weekly. First run: 3 min 52 s, inside the budget. It found a real bug (the inotify preflight was skipped when the node image had to be pulled first).

## Phase 4 — Fix mode and troubleshooting

Branch `phase-4-fix`.

- [ ] Mode plumbing: `setup.yaml`/`setup.sh` seeding, ownership labels, namespace and cluster-scoped cleanup by label, progress storage in `.k8slings.json`.
- [ ] `watch` behaviour for fix: poll the check while current, turn green on pass, no marker, "seeded, now fix it" message.
- [ ] `reset` re-seeds; `solution` prints `solution.sh`; `forbid:` rules in `check.yaml` (UID unchanged, image unchanged, resource not deleted-and-recreated).
- [ ] `TestSolutions` for fix: seed, check must fail, run `solution.sh`, check must pass; also a test that every forbid-shortcut variant fails.
- [ ] Troubleshooting topic: CrashLoopBackOff, ImagePullBackOff, Pending (resources, taints, PVC), broken selectors, failing probes, quota exceeded, RBAC denied, DNS/NetworkPolicy blocking; a multi-fault capstone (the prototype's 19 may live here).
- [ ] Docs for fix mode: what to do when you break the cluster for real (`cluster down && cluster up`, `reset`).

## Phase 5 — Advanced topics and add-ons

Branch per add-on.

- [ ] `metrics-server` add-on (pinned, `--kubelet-insecure-tls`); HPA exercises.
- [ ] Gateway add-on (Gateway API CRDs plus Traefik, after the spike confirms it serves both Ingress and Gateway API), any `extraPortMappings` create-time config, recreate-on-change prompt; Ingress and Gateway API exercises.
- [ ] Optional/extras: Helm basics (offline `helm template` check), PodDisruptionBudgets, ResourceQuota deep-dives, Pod Security Admission enforcement, custom resources and CEL validation rules (ties back to the assertion format).
- [ ] A separate add-on CI job.

## Phase 6 — CI, release and v1.0

- [ ] CI matrix finished: offline on every PR (Linux, Windows; Go 1.26/1.27/stable), cluster on every PR if within budget (else `main` + weekly + label), add-on job, macOS smoke weekly.
- [ ] Scheduled version-watch job (new kind release or node image).
- [ ] GoReleaser v2 config (linux, windows, darwin; amd64, arm64), release workflow on `v*` tags using the CHANGELOG section as release notes, `scripts/release-notes.sh`.
- [ ] README with install, quick start for offline-only and for cluster, the modes explained, demo GIFs; CONTRIBUTING finalized; issue templates finalized.
- [ ] Homebrew tap (optional; decide at the time).
- [ ] Docs consistency test green; every exercise proven in CI; v1.0.0 tag.

## Decided

Recorded 2026-11-04 after review of the first draft.

1. **License: Apache-2.0 for everything**, code and exercise content. `NOTICE` credits golings (Cristian Leonte) and the original mauricioabreu/golings, plus attribution for text and manifests adapted from `../learn/k8s-2-weeks`. `LICENSE` and `NOTICE` are written in Phase 1.
2. **Topology: 1 control plane + 2 workers by default**, `--single` for small machines.
3. **v1.0 scope: Phases 1-4** (offline, cluster and fix modes). HPA, Gateway and other Phase 5 topics ship as 1.x releases.
4. **Ingress/Gateway controller: Traefik**, provided the Phase 5 spike confirms it serves both Ingress and the Gateway API in kind; otherwise this question reopens.
5. **Lima:** the maintainer's Lima exposes a Docker-compatible socket, so Lima is supported on that path (Docker socket, `DOCKER_HOST` or the Docker template). Lima with containerd only stays best effort, untested.

6. **Kustomize: embed `sigs.k8s.io/kustomize/api`** so offline mode needs nothing installed. The Phase 0 size spike can still force a fallback to `kubectl kustomize` if the cost is unreasonable.
7. **Implicit cluster start: ask first.** The TUI prompts before creating the cluster (about 2 minutes, about 1.5 GB RAM); plain mode prints `k8slings cluster up` and exits non-zero. Never start silently.
8. **Repository:** private on GitHub (`cleonte/k8slings`) until the maintainer decides to open it.

Defaults taken for the rest, unless you object: same repo and module; `check.yaml` embedded only and hidden from the workspace; one target Kubernetes version at a time; fix-mode completion is sticky; `scripts/e2e.sh` deleted once `TestSolutions` covers cluster mode.

## Open questions

None at the moment. New questions found during the Phase 0 spikes or later phases are added here with a recommendation.
