# ROADMAP — k8slings

k8slings is a Rustlings-style course for Kubernetes, built the way [golings](https://github.com/cleonte/golings) is built. This file is the plan; `CLAUDE.md` is the architecture. Both are **design documents awaiting review**: no code changes until they are approved. Phase 0 is where the open decisions are made; the checkboxes in Phase 0 are mostly "decision recorded" rather than "code written".

Counts in this file are planning numbers. Once the consistency test exists (Phase 1), real counts live only between `<!-- count:... -->` markers and the test keeps them equal to `info.toml`.

## Where things stand

A working prototype exists (single `package main`, stdlib only, `go 1.22`, 19 exercises numbered 01–19, an e2e script that proves each exercise: untouched run is blocked by the marker, marker removed but still broken fails, solution passes). It applies manifests to a kind cluster and polls Go check functions. What it lacks compared with golings: a manifest (`info.toml`), topic structure, the TUI, `solution`, `doctor`, `verify` semantics, a solutions-tree test in `go test`, releases, docs, and CI. What it lacks for Kubernetes specifically: an offline mode, kubeconfig isolation (today it passes `--context` but `kind create cluster` still rewrites the user's kubeconfig and current-context), add-on handling, and a fix mode.

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

### The 19 prototype exercises

Proposed destination for each. This is based on titles and what I remember the checks doing; **confirm each one by reading its check in Phase 1** before moving it, and never weaken a check in the move.

| Prototype | Proposed mode | Reason |
|---|---|---|
| 01 pod basics | manifest | spec fields; keep a cluster variant out of scope |
| 02 env vars | manifest | prototype reads logs; assert the env instead, which removes a race |
| 03 labels/selectors | manifest | already an apply-time rejection, the schema/CEL step reproduces it |
| 04 scale and image | manifest | replicas and image tag |
| 05 rolling update | cluster | the rollout is the lesson |
| 06 service selector | cluster | endpoints are the lesson; keep the EndpointSlice readiness caveat |
| 07 configmap | manifest | prototype reads logs; assert volume/env wiring |
| 08 probes | manifest | probe fields (a cluster follow-up covers gating traffic) |
| 09 resources | manifest | requests/limits values |
| 10 job | manifest | apply-time rejection |
| 11 rbac (read) | cluster | `can-i` needs the API server |
| 12 secrets | manifest | wiring and types |
| 13 pvc storage | cluster | binding needs the provisioner; inotify caveat |
| 14 init container | cluster | ordering is runtime |
| 15 scheduling | cluster, `multinode` | needs real nodes |
| 16 networkpolicy | cluster | real blocked traffic |
| 17 rbac (write) | cluster | `can-i` |
| 18 quota/limits | cluster | admission behaviour |
| 19 broken stack | cluster | multi-fault capstone; may move to the end or into the fix track |

## Phase 0 — Decisions (this document)

- [ ] Review and approve CLAUDE.md and ROADMAP.md.
- [x] Decisions recorded under "Decided"; no open questions remain.
- [ ] **Modes** recorded: manifest, cluster, fix (CLAUDE.md "Exercise modes").
- [ ] **Assertion format** recorded: `check.yaml` with CEL; cluster steps vocabulary; named-Go-check escape hatch.
- [ ] **Cluster handling** recorded: owned kubeconfig, context pinning, container cross-check, topology, runtime support matrix.
- [ ] **Versions** recorded: Kubernetes 1.37, kind ≥ v0.33.0, node image digest, Go floor 1.26.
- [x] **License** decided: Apache-2.0 (see Decided).
- [ ] Spike (throwaway branch, results written here): evaluate 3 representative CEL assertions with `cel-go` against a parsed Deployment, including `all`, `has` and a cross-object `exists`; measure the binary-size cost.
- [ ] Spike: strict schema validation of the same Deployment with `santhosh-tekuri/jsonschema` versus kubeconform as a library; confirm `contianers:` is rejected and error messages are learner-friendly; measure embedded schema size for the kinds in the topic list. Record which is chosen.
- [ ] Spike: bring up the 3-node cluster with the pinned image and its own kubeconfig on Colima; measure idle RAM and time to ready; run the prototype's exercise 16 to confirm NetworkPolicy enforcement.
- [ ] Spike: Kustomize in-process (decided: embed the library): binary-size and dependency cost.

## Phase 1 — Skeleton from golings, offline mode, first 10 exercises

Branch `phase-1-skeleton`. Reuse golings code where it fits, keep Apache-2.0 headers and NOTICE, adapt rather than reinvent.

- [ ] `go.mod` at Go 1.26, module `github.com/cleonte/k8slings`; package layout from CLAUDE.md; cobra root with `init`, `run`, `hint`, `list`, `verify`, `reset`, `solution` (plain output first).
- [ ] `embed.go` with the two FS values (exercises, solutions); `info.toml` loader with validation (every path exists, names unique, modes valid, `requires` known).
- [ ] Marker handling (`# I AM NOT DONE`, `# TODO:`), `n`-style mark-done helper, workspace `init`/`reset` that never overwrites.
- [ ] Schema package with generated, embedded schemas (`scripts/gen-schemas.sh`), strict mode, learner-friendly error shortening.
- [ ] `check` package: `check.yaml` model, CEL evaluator, failure messages; unit tests for both.
- [ ] Manifest-mode runner.
- [ ] `TestSolutions` for manifest mode, plus manifest tests (every exercise has a solution and vice versa, hints and docs links present, no `metadata.namespace`, every kind used has an embedded schema).
- [ ] First 10 exercises: pods (5), labels and selectors (4) and namespaces (1 to start), with README, hint, docs link, TODO, solution.
- [ ] Migrate the prototype's manifest-destined exercises (01–04, 07–10, 12) once each check has been reviewed.
- [ ] Docs consistency test with `<!-- count:... -->` markers in README and ROADMAP.
- [ ] README, CHANGELOG (Keep a Changelog), CONTRIBUTING (adding an exercise, running tests, releasing), `.gitignore`, issue templates (bug, exercise problem, feature), `dependabot.yml` (grouped Go and Actions updates). `LICENSE` and `NOTICE` per the Phase 0 decision.
- [ ] CI: build/vet/test on Linux for Go 1.26, 1.27, stable; Windows job; formatting and vet.

## Phase 2 — TUI and tooling parity with golings

Branch `phase-2-tui`.

- [ ] Watcher package (debounced, sees atomic saves), copied and adapted.
- [ ] Bubble Tea v2 full-screen `watch`: progress, topic, result pane, re-run on save; keys `h s r n l c f q`; scrolling; `--plain` and automatic fallback when not a TTY.
- [ ] `l` list with filter by name or topic; `c` check-all (parallel, offline only for now); `f` full output.
- [ ] `doctor` (offline part first: workspace, schema version, embedded versions); `--verbose` flags.
- [ ] `solution` gating, `reset` confirmation, failure shortening for schema and CEL errors.
- [ ] `make smoke` and `scripts/smoke.sh`; macOS weekly smoke workflow; demo tapes for the README (vhs).

## Phase 3 — Cluster mode and its topics

Branch `phase-3-cluster`.

- [ ] `cluster` package: kind wrapper with pinned image, owned kubeconfig, context pinning, the single kubectl builder, container cross-check, refusal rules, and a test that no other code path execs kubectl.
- [ ] `cluster up | down | status`, `kubectl` pass-through, image preload, `--single`.
- [ ] Add-on registry skeleton (`requires` handling, skip messages, `list` locks); `multinode` topology.
- [ ] `doctor` cluster part: runtime detection (Docker Desktop, Colima, Lima, Linux), kind/kubectl versions and skew, node/core-pod health, inotify limits with the printed fix.
- [ ] Declarative cluster `check.yaml` steps (`wait`, `assert`, `endpoints`, `http` via API-server proxy, `exec`, `can_i`, `logs`) plus the named-Go-check escape hatch.
- [ ] Cluster-mode runner (namespace wipe/recreate, apply, poll, diagnostics, timeouts, global ceiling).
- [ ] `TestSolutions` for cluster mode behind `-tags cluster` or an env var; `make cluster-test`.
- [ ] Migrate the prototype's cluster-destined exercises (05, 06, 11, 13–19); retire `scripts/e2e.sh` once `TestSolutions` covers them.
- [ ] Topics: deployments and rollouts, services, probes (runtime), resources and quota, jobs and CronJobs, volumes and PVCs, StatefulSets, DaemonSets, scheduling.
- [ ] Remaining offline topics from the table (ConfigMaps/Secrets, Kustomize, security contexts, RBAC, network policies shape).
- [ ] Cluster CI job on every PR (path-filtered), with the time budget rule from CLAUDE.md.

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
