# CLAUDE.md — k8slings

> **Status: Phase 1 in progress.** The CLI skeleton, manifest mode (schema validation and CEL checks), `init run hint list verify reset solution`, `watch` (full-screen and `--plain`), and the first exercises are built. Everything marked *(target)* is still design: `doctor`, cluster and fix modes, `cluster` commands and add-ons. The earlier prototype lives in `prototype/` (its own Go module) until its exercises are migrated. See ROADMAP.md for what is done.

## What this project is

k8slings is "rustlings for Kubernetes": an interactive CLI that teaches Kubernetes through small broken exercises the learner fixes. It is the sibling of [golings](https://github.com/cleonte/golings) (itself a fork of [mauricioabreu/golings](https://github.com/mauricioabreu/golings)), and it deliberately has the same CLI shape, manifest, solutions tree, TUI, CI discipline and docs. When in doubt about how something should work, read how golings does it (a clone lives at `../golings`) and copy that, adapting only what Kubernetes forces to be different.

What Kubernetes forces to be different is that an exercise is not always a file that compiles. There are three exercise modes (below), and only one of them needs a cluster. Most beginner exercises are offline, so a learner can practise for 20 minutes without installing Docker.

Source material for new exercises: `../learn/k8s-2-weeks/` (day-01 to day-10, `extras/`, `core-advanced/`, `ckad-gaps/`). That repository is MIT for code and CC BY 4.0 for text, so anything adapted from it needs attribution in `NOTICE`.

## Architecture *(target)*

A Go CLI, module `github.com/cleonte/k8slings`, Go floor 1.26 (same as golings), built on cobra and the Bubble Tea v2 / Lip Gloss v2 / Bubbles v2 stack. The prototype was "stdlib only"; that rule is dropped on purpose, because the manifest mode needs a YAML parser, a JSON Schema validator and a CEL evaluator that nobody should write by hand. Every new dependency still needs a one-line justification in the PR that adds it.

```
k8slings/                      Go module root
  embed.go                     embeds info.toml, exercises/ (FS) and solutions/ (Solutions, never written by init)
  info.toml                    the manifest, in learning order
  k8slings/                    the CLI package (main, as golings/golings.go)
    cmd/                       cobra commands
    exercises/                 manifest loading, runner, failure shortening, mode dispatch
    check/                     check.yaml model and the CEL evaluator (shared by all three modes)
    schema/                    embedded Kubernetes JSON schemas and the strict validator
    cluster/                   kind wrapper, kubeconfig isolation, add-ons, kubectl runner
    tui/ ui/ watcher/ workspace/ doctor/ fixtures/     as in golings
  exercises/<topic>/<name>/    learner-facing files (see "Exercise layout")
  solutions/<topic>/<name>/    verified solution, mirrors exercises/
  schemas/                     generated, embedded Kubernetes schemas (see "Schemas")
  addons/                      pinned add-on manifests (metrics-server, Gateway API, a controller)
  scripts/                     smoke.sh, release-notes.sh, gen-schemas.sh, e2e helpers
  .github/                     workflows, issue templates, dependabot.yml
```

Embedding follows golings: `info.toml` and `exercises/` go in `FS`; `solutions/` goes in a separate `Solutions` FS so `init` can never write it. `check.yaml` files are embedded but **not** written to the learner's workspace, so the assertions are not a spoiler; failure messages say what is expected without giving the answer. The embedded copy is also what `reset` restores from.

### Commands *(target)*

Same set as golings, plus the cluster commands:

| Command | Behaviour |
|---|---|
| `init [dir]` | Write the exercises and `info.toml` into a workspace. Never overwrites existing work. |
| `watch` | Full-screen Bubble Tea view. `--plain` for line mode, and it falls back to plain when stdin or stdout is not a terminal. |
| `run [name]` | Run one exercise, or the current one. Accepts a name or a number. |
| `hint [name]` | Print the hint and the docs link. |
| `list` | Exercises with state, grouped by topic, with progress. |
| `verify` | Run every exercise in order and stop at the first that is not done (CI for learners). |
| `reset <name>` | Restore one exercise from the embedded copy. In fix mode it also re-seeds the cluster setup. |
| `solution <name>` | Hidden while the exercise is pending; `--reveal` to look anyway. |
| `doctor` | Check the environment (see "Doctor"). |
| `cluster up\|down\|status` | Manage the one kind cluster (see "Cluster handling"). `cluster addon install\|list`. |
| `kubectl ...` | Pass-through to kubectl with the k8slings kubeconfig and context forced. The only kubectl a learner needs in fix mode. |

Keys in the TUI are the golings keys: `h` hint, `s` solution (asks first), `r` reset (asks first), `n` next, `l` list, `c` check all, `f` full output, `q` quit. Check-all runs offline exercises everywhere and cluster/fix exercises only if the cluster is up; the rest are listed as skipped with the reason.

### Manifest: `info.toml` *(target)*

Every exercise is listed, in learning order. Order matters; do not append at the end by habit.

```toml
[[exercises]]
name    = "deployments3"
path    = "exercises/deployments/deployments3"     # a directory, not a file (fix mode has several files)
topic   = "deployments"
mode    = "manifest"                               # manifest | cluster | fix
hint    = "The selector and the pod template labels must agree."
docs    = "https://kubernetes.io/docs/concepts/workloads/controllers/deployment/"
k8s_version = "1.37"                               # optional, like go_version: skipped on a cluster/schema older than this
requires    = []                                   # optional add-ons/topology, see "Add-ons"
timeout     = "60s"                                # cluster and fix modes only; default per mode
```

`k8s_version` plays the role of `go_version`. `requires` plays the role of `race = true`: an exercise whose environment is not available is skipped with a message that says how to get it, never failed.

### Exercise layout *(target)*

All modes: `README.md` (what to learn, a few lines), and a `hint` + `docs` in `info.toml`. The learner edits `exercise.yaml` (manifest and cluster) and sees a `# TODO:` comment at the fix site; the marker is `# I AM NOT DONE` on the first line (YAML) exactly like golings' `// I AM NOT DONE`. `watch` and `run` refuse to report success while the marker is present; `n` removes it once the exercise passes.

```
exercises/<topic>/<name>/
  README.md            learner-facing lesson
  exercise.yaml        manifest + cluster modes: the broken manifest(s), may be multi-document
  check.yaml           embedded only, not written by init
  setup.yaml, ...      fix mode only: see below
solutions/<topic>/<name>/
  exercise.yaml        working version, no marker, no TODO     (manifest + cluster)
  solution.sh          kubectl script                           (fix)
```

### Schemas *(target)*

Manifest mode validates against the real Kubernetes OpenAPI schema for the **target Kubernetes version**, embedded in the binary. Nothing is downloaded at runtime. Validation is strict: unknown fields are errors, which is what catches `contianers:`. The source is the `v<version>-standalone-strict` set of [yannh/kubernetes-json-schema](https://github.com/yannh/kubernetes-json-schema) (the schemas kubeconform uses); `scripts/gen-schemas.sh` copies only the kinds that exercises use, to keep the binary small, and a test fails if an exercise uses a kind that is not in the embedded set. If that set ever lags a Kubernetes release, the script falls back to converting `api/openapi-spec` from `kubernetes/kubernetes` at the release tag. CRDs (Gateway API) are taken from the corresponding release's CRD files. The validator is `santhosh-tekuri/jsonschema` (to be confirmed in the Phase 1 spike against embedding kubeconform as a library; the choice must not change the check.yaml format).

## Exercise modes

### manifest — offline, no cluster

The learner fixes `exercise.yaml`. The runner (1) refuses if the marker is present, (2) parses the YAML (multi-document allowed), (3) validates every object against the embedded strict schema, (4) evaluates the assertions in `check.yaml`. No Docker, no kind, no kubectl. This is the default mode for anything whose correctness can be decided from the object itself, and should hold most beginner exercises: replicas, selectors and labels, probes, resources, env, volumes, RBAC rules, NetworkPolicy shape, Kustomize output.

The assertion format is `check.yaml` with [CEL](https://cel.dev/) expressions. CEL is what Kubernetes itself uses in ValidatingAdmissionPolicy and CRD validation rules, so learners meet a real Kubernetes idiom, and it handles cross-field checks that JSONPath cannot (selector vs template labels). Each expression is a boolean over `object` (the selected object); `objects` is the full list for cross-object checks.

```yaml
schema: strict                         # strict (default) | lenient | off
objects:
  - select: { kind: Deployment }       # exactly one match required, else a clear message
    assert:
      - expr: object.spec.replicas == 3
        message: "The Deployment should run 3 replicas."
      - expr: object.spec.selector.matchLabels.all(k, object.spec.template.metadata.labels[k] == object.spec.selector.matchLabels[k])
        message: "The selector must match the pod template labels."
      - expr: object.spec.template.spec.containers.all(c, has(c.readinessProbe))
        message: "Every container needs a readiness probe."
  - select: { kind: Service }
    assert:
      - expr: objects.exists(d, d.kind == 'Deployment' && object.spec.selector.all(k, d.spec.template.metadata.labels[k] == object.spec.selector[k]))
        message: "The Service selector matches no Deployment."
```

Trade-offs considered: JSONPath is simpler to read but cannot express "all keys of A are in B"; Rego/OPA is a big dependency and a second language; Go test files per exercise would make contributors write Go and would put the answer in the learner's workspace. CEL is the middle: one small, well-known dependency (`google/cel-go`), declarative, safe (no side effects, bounded cost).

Rules:
- Messages are written for the learner and never contain the answer. Say what is wrong, not what to type.
- Every assertion is idempotent and pure. A manifest check that depends on the clock or the network is a bug.
- Do not assert on defaulted fields the API server would fill in; assert only what the learner wrote.
- Offline does not mean unvalidated: if a mistake would be rejected by the API server (wrong type, unknown field, missing required field), the schema step must catch it, and an exercise based on such a rejection uses `schema: strict` as the "check" with an empty `objects`.

### cluster — apply to a real cluster

The runner applies `exercise.yaml` to a fresh namespace in the k8slings cluster, waits for real conditions, asserts, and cleans up. Use this mode only when the thing being taught is runtime behaviour: rollouts, probes actually gating traffic, a Service getting endpoints, a Job completing, PVC binding, scheduling, NetworkPolicy blocking traffic, RBAC `can-i`, quota admission.

Flow, inherited from the prototype: marker gate, cluster reachable, wipe and recreate the namespace `k8slings-<name>`, `kubectl apply -n <ns>`, then poll the check every 2 s until it passes or the timeout expires, then print diagnostics (pod states, events, last logs) if it did not. Exercise YAML never sets `metadata.namespace`.

`check.yaml` for cluster mode is declarative too, using the same CEL against **live** objects, with a small step vocabulary:

```yaml
steps:
  - wait: { for: rollout, resource: deployment/web, timeout: 90s }
  - wait: { for: condition=Ready, resource: pod, selector: app=web }
  - assert:
      resource: service/web
      expr: object.status.loadBalancer == null     # any CEL over the live object
  - endpoints: { service: web, min: 2 }            # EndpointSlice ready addresses
  - http: { service: web, port: 80, path: /, status: 200, body_contains: "nginx" }
  - exec: { pod: client, args: ["wget","-qO-","-T","3","http://server"], ok: true }
  - exec: { pod: intruder, args: ["wget","-qO-","-T","3","http://server"], ok: false }
  - can_i: { as: sa/reader, verb: get, resource: pods, ok: true }
  - logs: { pod: app, contains: "ready" }
```

`http` goes through the API-server service proxy (`kubectl get --raw /api/v1/namespaces/<ns>/services/<name>:<port>/proxy/`), so no port-forward process, no host port, no firewall prompt, and it works the same on Colima, Lima and Docker Desktop. Traffic-policy questions (NetworkPolicy) must not use it, because the proxy bypasses pod-to-pod policy; those use `exec` from a client pod inside the namespace. Ingress/Gateway exercises are the one place that needs host reachability; see Add-ons.

Why declarative and not a Go `Check` func per exercise (as in the prototype's `registry`): a declarative file is reviewable without reading Go, shares one evaluator and one set of timeouts, produces uniform failure output, and lets the same file drive fix mode. The cost is a fixed vocabulary. The escape hatch is a **named Go check**: `check.yaml` may contain `go: <name>`, resolved in a registry in `k8slings/check/named.go`. It is reserved for checks the vocabulary cannot express (parsing log content, arithmetic on quota status) and every use needs a sentence in the PR on why the vocabulary was not enough. Named checks follow the same purity rules.

All checks are **read-only and idempotent**: they are polled repeatedly. They must never mutate the cluster. A step that fails reports the first failing step with expected-versus-actual, not a dump.

### fix — troubleshooting with live state *(target)*

CKA-style practice. The tool brings the cluster to a broken state; the learner repairs it with `kubectl` (through `k8slings kubectl` or their own kubectl with the k8slings kubeconfig); the check asserts the end state. There is **no file to edit**, so the file-based mechanics change:

- Files: `README.md` (the symptom as a ticket: "the web app returns 503, fix it, do not delete the Deployment"), `setup.yaml` (the broken state, multi-document), optional `setup.sh` for things that are not plain manifests (a `kubectl taint`, a scale), `check.yaml` (as cluster mode, plus optional `forbid:` rules), and in `solutions/` a `solution.sh`.
- No marker. The "done" signal is the check passing: in `watch` a fix exercise is polled every 2 s while it is current, and when the check passes it turns green and `n` moves on. Outside watch, `k8slings verify <name>` or `run <name>` runs the check once. Completion is stored in the progress file (`.k8slings.json` in the workspace), because there is no file to mutate.
- `reset <name>` deletes the namespace and re-runs the seed. `run` on a fix exercise never reseeds by itself (it would wipe the learner's work); the first `run`/`watch` on a pending exercise seeds it, and the TUI says "seeded, now fix it".
- "Solution" means `solution.sh`: a kubectl script, run with `KUBECONFIG` and `NS` set. `solution <name>` prints it (hidden while pending). It is the CI proof, see below.
- Shortcuts are blocked by `forbid:` assertions in `check.yaml`, such as "the Deployment UID must be unchanged" or "the image must still be nginx:1.27", so deleting and recreating things does not pass. The check must pass for the legitimate end state and fail for the shortcut.
- Safety: fix exercises only touch their own namespace, plus cluster-scoped objects that the seed creates and `reset`/cleanup removes by label (`k8slings.io/exercise=<name>`). A fix exercise that taints a node must untaint it in cleanup; those need `requires = ["multinode"]` so the control plane is never the victim.

## Cluster handling and safety *(target; the prototype only partly does this)*

Hard rule: **k8slings only ever talks to its own cluster, and never to the user's current context.**

- `kind create cluster` by default writes into `~/.kube/config` and switches `current-context`. That violates the rule, so `cluster up` runs `kind create cluster --name k8slings --kubeconfig <owned file>`. The owned file lives under the OS user config dir (`os.UserConfigDir()/k8slings/kubeconfig`). It is never merged into `~/.kube/config`.
- Every `kubectl` invocation passes `--kubeconfig <owned file> --context kind-k8slings` and runs with `KUBECONFIG` removed from its environment. There is one function that builds kubectl command lines; a unit test fails if any other code path execs `kubectl`.
- Before any cluster or fix exercise runs, the runner verifies that the API server in the owned kubeconfig is the one published by the `k8slings-control-plane` container (compare the server port with `docker port`). Any mismatch, a missing container, or a context name other than `kind-k8slings` is a hard refusal with an explanation. There is no flag to override it.
- Cluster-scoped changes are limited to the k8slings cluster by construction, but exercises still prefer namespaced objects. Cluster-scoped objects (ClusterRole, StorageClass, PriorityClass, taints) are labelled and removed by label.
- `cluster up` is idempotent and reuses an existing cluster. `cluster down` deletes only the cluster named `k8slings` and the owned kubeconfig.

### Version pinning

The target Kubernetes version is the newest that kind supports, **checked, not assumed**: kind v0.33.0 (released 2026-08-26) ships `kindest/node:v1.37.0@sha256:a1ed56cf…580ae5` as its default image; Kubernetes v1.38 is still alpha. `cluster up` passes the image **with its digest** (`--image`), so the cluster is identical on every machine. The full digest lives in one Go constant next to the minimum kind version. `doctor` warns when kind is older than the version the image was published with, and when `kubectl` is more than one minor version away from the cluster. Bumping Kubernetes is a deliberate PR: new node image, regenerated schemas, `k8s_version` reviewed, all exercises re-proven. A scheduled CI job opens an issue when a newer kind release lists a newer default image (Dependabot cannot see this).

### Topology

Node topology is fixed at cluster creation, so it is a Phase 0 decision rather than a per-exercise one: DaemonSet, taint, anti-affinity and spread exercises are meaningless on one node. Default: **one control-plane plus two workers** (about 1.2–1.5 GB of RAM idle, fine inside the 6 GB Colima VM the prototype already documents). `cluster up --single` creates one node for small machines; exercises with `requires = ["multinode"]` are skipped with a message. The create-time kind config is generated from the add-ons the user asked for, so enabling one that needs `extraPortMappings` means recreating the cluster, and `cluster up` says so instead of silently doing it.

### Container runtime

Docker Desktop and Docker on Linux are the supported baseline. macOS with Colima (Docker runtime, `colima start --vm-type vz --cpu 4 --memory 6`) is the primary tested macOS path, since that is the maintainer's setup. Lima works when it exposes a Docker-compatible socket (the Docker template, or `DOCKER_HOST` pointing at it); Lima with only containerd/nerdctl uses kind's experimental provider and is **best effort, not tested in CI**. `doctor` reports which runtime and socket it found and prints the exact fix for the common failure (Docker not running, wrong `DOCKER_HOST`, the inotify limits below).

### Doctor *(target)*

`k8slings doctor` checks: Go is irrelevant at run time (the binary is static), but it checks Docker or a compatible runtime is reachable and reports its kind; `kind` present and at least the pinned minimum version; `kubectl` present and within one minor version of the cluster; the cluster exists, is the right one, its nodes are Ready and core pods (CoreDNS, kube-proxy, kindnet, local-path-provisioner) are Running; the inotify limits on Colima/Lima VMs, **before** `cluster up` (measured: Lima's default `max_user_instances = 128` makes a 3-node `kind create` report success while workers never join and kube-proxy crashes); which add-ons are installed. `cluster up` itself must verify all nodes Ready and kube-proxy Running rather than trust kind's exit code. Offline mode needs none of this, so `doctor` separates "offline ready" from "cluster ready" and exits 0 when offline is ready, with a clear note about what the cluster part is missing.

## Add-ons *(target)*

Default kind has no metrics-server, no ingress or Gateway controller. **NetworkPolicy needs no add-on:** kind's default CNI (kindnet) has enforced NetworkPolicy via `kube-network-policies` since v0.24, and the prototype's NetworkPolicy exercise tests real blocked traffic. (Phase 0 task: confirm on the pinned image by running that exercise, and record the result in ROADMAP.)

An exercise declares what it needs in `info.toml`, the way golings declares `go_version` and `race`:

```toml
requires = ["multinode", "metrics-server"]    # topology: multinode; add-ons: metrics-server, gateway
```

The add-on registry (`k8slings/cluster/addons.go`) has, per add-on: name, pinned manifest in `addons/`, a readiness check, and whether it needs create-time cluster config. `cluster up --addons metrics-server,gateway` installs them; `cluster addon install <name>` installs later when no recreate is needed. When `run` hits an exercise whose `requires` is not satisfied it skips with one line: `needs metrics-server: run k8slings cluster addon install metrics-server`. `list` shows the lock, `check all` reports skipped, `verify` treats skipped as not-done-but-not-failed. Planned add-ons: `metrics-server` (HPA; needs `--kubelet-insecure-tls` on kind), `gateway` (Gateway API CRDs plus one implementation). **ingress-nginx is retired** (the Kubernetes project announced it in November 2025 and stopped maintenance in March 2026), so it is not an option; the chosen controller is Traefik (ROADMAP.md, "Decided"), pending a Phase 5 spike that it serves both Ingress and the Gateway API. Manifests are pinned to exact versions and live in the repo, not fetched from `main` at install time.

## Speed

Cluster exercises must check quickly. Rules: reuse one cluster; one namespace per exercise; poll every 2 s with a per-exercise timeout (default 60 s cluster, 120 s fix) and a global ceiling; delete the namespace with `--wait=false` followed by a wait for the *next* run's namespace only; pre-load the handful of images exercises use (`busybox`, `nginx`, `registry.k8s.io/pause`, a tiny curl image) with `kind load` during `cluster up`, so no exercise depends on a registry pull; prefer images that are already in the node image. An exercise whose solution takes more than about 30 s on CI hardware needs a reason.

## CI *(target)*

- **Every PR, no cluster:** `go build`, `go vet`, `gofmt -l`, unit tests, and `TestSolutions` for every **manifest** exercise (unsolved fails, solution passes; no solution without an exercise; no exercise without a solution). Also the manifest consistency tests and the docs-counts test. Matrix: Go 1.26, 1.27, stable, on Linux. Windows and the `go install` smoke test as in golings, since offline mode must work on Windows; macOS smoke weekly.
- **Cluster job (kind in GitHub Actions):** brings up the pinned cluster, runs `TestSolutions` for cluster and fix exercises: seed or apply, the check **must fail** (with the short `BROKEN_TIMEOUT`), then the solution (`exercise.yaml` or `solution.sh`) is applied and the check **must pass**. The prototype's e2e takes about 2.5 minutes for its 19 exercises at four in parallel on a laptop; a hosted runner adds roughly 1–2 minutes for cluster creation and image preload. That is cheap enough to run on **every PR** (path-filtered to exercises, solutions, the Go sources and the add-on manifests), sharded by exercise if it grows past about 10 minutes. If it ever exceeds 15 minutes even sharded, it moves to `main` + weekly + a PR label, and ROADMAP records that decision.
- **Add-on job:** the exercises that need metrics-server or the gateway controller run in a separate job so a flaky add-on does not block basic cluster exercises.
- A consistency test keeps docs honest: README and ROADMAP carry markers like `<!-- count:exercises -->`, and a test compares them with `info.toml` (total, per mode, per topic). **Do not hardcode exercise counts in this file** (or anywhere outside the marked places).

## Rules for exercises (all modes)

- Each exercise is small and single-concept. A `README.md` says what to learn in a few lines; the YAML has `# I AM NOT DONE` on line 1 and a `# TODO:` comment at the fix site, one sentence, without giving the answer away (the solution does not keep either).
- Idiomatic, current Kubernetes only: `apps/v1`, `networking.k8s.io/v1`, no removed APIs, no `kubectl run --generator`, no Endpoints where EndpointSlices are meant, pinned image tags (never `latest`), `securityContext` shown in the secure style where it is not the topic.
- The unsolved version must fail every time, the same way. Failures should come from the exercise, not from a broken cluster.
- Never weaken a check just to make a solution pass. If bug, solution and check disagree, fix the exercise.
- Every exercise has a verified solution in `solutions/`, mirroring `exercises/`. Order matters: put new exercises where they belong in the learning progression.
- Pick the cheapest mode that can decide correctness. Promote manifest to cluster only if runtime behaviour is the lesson.
- Cluster and fix exercise YAML never sets `metadata.namespace` (k8slings picks it). Manifest exercises may, when the namespace is the lesson.
- No behaviour that depends on a registry rate limit, the wall clock or the order of API list results.
- Windows: manifest mode and all paths in `info.toml` use forward slashes; no file permissions in tests.

## Commands for contributors *(target)*

```sh
make test            # build, vet, unit tests, offline TestSolutions (go test -short skips the slow ones)
make cluster-test    # needs docker + kind; brings up the pinned cluster and proves cluster/fix exercises
make smoke           # installs into a temp dir, runs init, list, doctor, run on the first exercise
make watch / make run
```

## The prototype (what is in the repo now)

The current code is a working single-package `package main` (about 1,000 lines, stdlib only, `go 1.22`, module `github.com/cleonte/k8slings`) with 19 cluster-style exercises, `scripts/e2e.sh`, and a registry of Go `Check` funcs in `exercises.go`. It will be migrated, not discarded: the engine moves into the package layout above, the 19 exercises move to the `exercises/<topic>/<name>/` tree and `info.toml`, and each becomes manifest or cluster mode by what it actually tests (Phase 1 and 3 in ROADMAP.md). `scripts/e2e.sh` stays as a development convenience until `TestSolutions` replaces it. Facts worth keeping while it is migrated:

- Per-exercise namespace is `k8slings-<id>`. Before each check every namespaced resource type from `kubectl api-resources` is deleted (no kind list to maintain) and `run` waits for `sa/default` to reappear, because bare Pods are rejected until then.
- The "still broken" e2e case uses a short timeout (`BROKEN_TIMEOUT`, default 20 s) rather than the exercise's; the solution run uses the real one.
- Exercises 03 and 10 are broken at apply time (API rejection); `Run` returns the apply error, so the "still broken" case fails fast. These are natural manifest-mode exercises, but note that the rejection comes from the API server's semantic validation, not from the OpenAPI schema, so it must be reproduced with a CEL assertion in `check.yaml` (a Deployment selector must match its template labels; a Job's `restartPolicy` must be `OnFailure` or `Never`). The schema alone catches only type and unknown-field errors.
- Exercise 06 reads EndpointSlices with the jsonpath filter `?(@.conditions.ready==true)`; this depends on the kind node image, so re-prove it on every Kubernetes bump.
- Exercises 01 and 08 pull `nginx`; the first run is slow and timeouts may need tuning (the image preload in `cluster up` addresses this).
- Exercises 02 and 07 read pod logs right after Running; a race is possible and polling must absorb it.
- `Check` functions and namespace cleanup in `runner.go` are the likeliest sources of bugs.
- **Docker VM inotify limits** (Colima/Lima): if kube-proxy, coredns or local-path-provisioner crash with "too many open files", run `colima ssh -- sudo sysctl -w fs.inotify.max_user_instances=512 fs.inotify.max_user_watches=524288` and restart those pods. Symptoms: DNS "bad address" in the exercises that use Service DNS (16, 19), PVC stuck Pending (13). `doctor` must detect and print this.

## Licensing and attribution

Code copied or adapted from golings stays under golings' **Apache-2.0** license, with a `NOTICE` that credits both golings (Cristian Leonte) and the original mauricioabreu/golings, and `LICENSE` kept intact. k8slings' own new code and exercise content are also **Apache-2.0** (decided 2026-11-04), so the repository has a single license. `LICENSE` and `NOTICE` are written in Phase 1. Text and manifests adapted from `../learn/k8s-2-weeks` (MIT code, CC BY 4.0 text) need attribution in `NOTICE`.

## Working style

- Work on a branch per roadmap step, never straight on `main`, and open a PR per step. Branch names like `phase-1-skeleton`.
- Small commits, one logical change each. Run `make test` before every commit; for anything touching cluster, add or run `make cluster-test` for the exercises affected.
- Tests stay green on every commit. A new exercise without a solution fails the build on purpose.
- Do not rewrite an exercise's intent when modernising; fix idioms and wording only.
- Commit messages and PR descriptions carry no `Co-Authored-By` or "Generated with" trailers.
- When unsure about a format, read the golings code (`../golings/golings/exercises/`), then an existing k8slings example, then follow them.
- Changes to this file and ROADMAP.md that alter a decision go in their own commit, with the reason.
