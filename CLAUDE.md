# CLAUDE.md — k8slings

> **Status: usable and proven; two pieces are still to build or to grow.** Manifest mode, cluster mode (the isolated kind cluster, declarative check steps, the runner, image preload), `watch` (full-screen and `--plain`), `doctor` (offline and cluster), <!-- count:exercises -->99<!-- /count --> exercises in <!-- count:topics -->21<!-- /count --> topics, and the release setup (GoReleaser, version watch) are built and proven in CI, including on a real kind cluster. **Fix mode** (Phase 4) is built, with seventeen troubleshooting exercises and a capstone; more of them, and cluster-scoped setups, are next. Sections marked *(target)* describe what is not built yet: the **add-on registry** (Phase 5). See ROADMAP.md for the plan.

## What this project is

k8slings is "rustlings for Kubernetes": an interactive CLI that teaches Kubernetes through small broken exercises the learner fixes. It is the sibling of [golings](https://github.com/cleonte/golings) (itself a fork of [mauricioabreu/golings](https://github.com/mauricioabreu/golings)), and it deliberately has the same CLI shape, manifest, solutions tree, TUI, CI discipline and docs. When in doubt about how something should work, read how golings does it (a clone lives at `../golings`) and copy that, adapting only what Kubernetes forces to be different.

What Kubernetes forces to be different is that an exercise is not always a file that compiles. There are three exercise modes (below), and only one of them needs a cluster. Most beginner exercises are offline, so a learner can practise for 20 minutes without installing Docker.

Source material for new exercises: `../learn/k8s-2-weeks/` (day-01 to day-10, `extras/`, `core-advanced/`, `ckad-gaps/`). That repository is MIT for code and CC BY 4.0 for text, so anything adapted from it needs attribution in `NOTICE`.

## Architecture

A Go CLI, module `github.com/cleonte/k8slings`, Go floor 1.26 (same as golings), built on cobra and the Bubble Tea v2 / Lip Gloss v2 / Bubbles v2 stack. The first prototype was "stdlib only"; that rule was dropped on purpose, because the manifest mode needs a YAML parser, a JSON Schema validator and a CEL evaluator that nobody should write by hand. Every new dependency still needs a one-line justification in the PR that adds it.

```
k8slings/                      Go module root
  embed.go                     embeds info.toml, exercises/ (FS) and solutions/ (Solutions, never written by init)
  info.toml                    the manifest, in learning order
  k8slings/                    the CLI package (main, as golings/golings.go)
    cmd/                       cobra commands
    exercises/                 manifest (info.toml), state and markers, the manifest-mode and cluster-mode runners
    check/                     check.yaml model, the CEL evaluator (with quantity()) and the cluster steps
    schema/                    embedded Kubernetes JSON schemas and the strict validator
    cluster/                   kind wrapper, kubeconfig isolation, the only code that runs programs (exec.go), kubectl, namespace reset, preload, health
    tui/ watcher/ workspace/ doctor/     the full-screen watch, file watching, init and reset, the two halves of doctor
  exercises/<topic>/<name>/    learner-facing files (see "Exercise layout")
  solutions/<topic>/<name>/    verified solution, mirrors exercises/
  schemas/                     generated, embedded Kubernetes schemas (see "Schemas")
  addons/                      (Phase 5, not there yet) pinned add-on manifests: metrics-server, Gateway API, Traefik
  scripts/                     smoke.sh, gen-schemas.sh, release-notes.sh, version-watch.sh
  misc/ demo*.tape             the README demo GIFs and the vhs tapes that record them
  .goreleaser.yaml             release builds; .github/ has the workflows, issue templates and dependabot.yml
```

Embedding follows golings: `info.toml` and `exercises/` go in `FS`; `solutions/` goes in a separate `Solutions` FS so `init` can never write it. `check.yaml` files are embedded but **not** written to the learner's workspace, so the assertions are not a spoiler; failure messages say what is expected without giving the answer. The embedded copy is also what `reset` restores from.

### Commands

Same set as golings, plus the cluster commands:

| Command | Behaviour |
|---|---|
| `init [dir]` | Write the exercises and `info.toml` into a workspace. Never overwrites existing work. |
| `watch` | Full-screen Bubble Tea view. `--plain` for line mode, and it falls back to plain when stdin or stdout is not a terminal. |
| `run next` / `run <name>` | Run the first exercise that is not done, or a named one. |
| `hint [name]` | Print the hint and the docs link. |
| `list` | Exercises with state, grouped by topic, with progress. |
| `verify` | Run every exercise in order and stop at the first that is not done (CI for learners). |
| `reset <name>` | Restore one exercise from the embedded copy. (Fix mode, when it exists, also re-seeds the cluster setup.) |
| `solution <name>` | Hidden while the exercise is pending; `--reveal` to look anyway. |
| `doctor` | Check the environment (see "Doctor"). |
| `cluster up\|down\|status` | Manage the one kind cluster (see "Cluster handling"). `up --single` for one node. (`cluster addon install\|list` arrives with the add-ons, Phase 5.) |
| `kubectl ...` | Pass-through to kubectl with the k8slings kubeconfig and context forced. The only kubectl a learner needs in the cluster exercises, and in fix mode. |

Keys in the TUI are the golings keys: `h` hint, `s` solution (asks first), `r` reset (asks first), `e` edit (opens the exercise file, or the ticket of a fix exercise, in `$VISUAL` or `$EDITOR`, and waits; the key is only shown when an editor is set; the command is built in `cluster/exec.go` because that is the only file that may import `os/exec`), `n` next, `l` list, `c` check all, `f` full output, `q` quit. Check-all runs offline exercises everywhere and cluster/fix exercises only if the cluster is up; the rest are listed as skipped with the reason.

### Manifest: `info.toml`

Every exercise is listed, in learning order. Order matters; do not append at the end by habit.

```toml
[[exercises]]
name    = "deployments3"
path    = "exercises/deployments/deployments3"     # a directory, not a file (fix mode has several files)
topic   = "deployments"
mode    = "manifest"                               # manifest | cluster | fix
hint    = "The selector and the pod template labels must agree."
docs    = "https://kubernetes.io/docs/concepts/workloads/controllers/deployment/"
k8s_version = "1.37"                               # optional, like go_version (parsed, not enforced yet: there is one target version)
requires    = []                                   # optional add-ons/topology, see "Add-ons"
timeout     = "60s"                                # cluster and fix modes only; default per mode
```

`k8s_version` plays the role of `go_version`. `requires` plays the role of `race = true`: an exercise whose environment is not available is skipped with a message that says how to get it, never failed. Today `requires = ["multinode"]` works (skipped on `cluster up --single`); add-on names are skipped with "needs the X add-on, which is not available yet" until Phase 5.

### Exercise layout

All modes: `README.md` (what to learn, a few lines), and a `hint` + `docs` in `info.toml`. The learner edits `exercise.yaml` (manifest and cluster) and sees a `# TODO:` comment at the fix site; the marker is `# I AM NOT DONE` on the first line (YAML) exactly like golings' `// I AM NOT DONE`. `watch` and `run` refuse to report success while the marker is present; `n` removes it once the exercise passes.

```
exercises/<topic>/<name>/
  README.md            learner-facing lesson
  exercise.yaml        manifest + cluster modes: the broken manifest(s), may be multi-document (a fix exercise has none)
  check.yaml           embedded only, not written by init
  setup.yaml           fix mode only: the broken state, embedded only like check.yaml
  setup-2.yaml ...     fix mode only, optional: further steps applied after the previous one has rolled out (see Seeding)
solutions/<topic>/<name>/
  exercise.yaml        working version, no marker, no TODO     (manifest + cluster)
  solution.sh          kubectl script that repairs it           (fix)
  shortcut-*.sh        ways to cheat that the check must refuse (fix)
```

### Schemas

Manifest mode validates against the real Kubernetes OpenAPI schema for the **target Kubernetes version**, embedded in the binary. Nothing is downloaded at runtime. Validation is strict: unknown fields are errors, which is what catches `contianers:`. The source is the `v<version>-standalone-strict` set of [yannh/kubernetes-json-schema](https://github.com/yannh/kubernetes-json-schema) (the schemas kubeconform uses); `scripts/gen-schemas.sh` copies only the kinds that exercises use, to keep the binary small, and a test fails if an exercise uses a kind that is not in the embedded set. If that set ever lags a Kubernetes release, the script falls back to converting `api/openapi-spec` from `kubernetes/kubernetes` at the release tag. CRDs (Gateway API) are taken from the corresponding release's CRD files. The validator is `santhosh-tekuri/jsonschema/v6`; the errors it produces are shortened for the learner (`spec.template.spec: additional properties 'contianers' not allowed`).

## Exercise modes

### manifest — offline, no cluster

The learner fixes `exercise.yaml`. The runner (1) refuses if the marker is present, (2) parses the YAML (multi-document allowed), (3) validates every object against the embedded strict schema, (4) evaluates the assertions in `check.yaml`. No Docker, no kind, no kubectl. This is the default mode for anything whose correctness can be decided from the object itself, and should hold most beginner exercises: replicas, selectors and labels, probes, resources, env, volumes, RBAC rules, NetworkPolicy shape, Kustomize output.

The assertion format is `check.yaml` with [CEL](https://cel.dev/) expressions. CEL is what Kubernetes itself uses in ValidatingAdmissionPolicy and CRD validation rules, so learners meet a real Kubernetes idiom, and it handles cross-field checks that JSONPath cannot (selector vs template labels). Each expression is a boolean over `object` (the selected object); `objects` is the full list for cross-object checks; `quantity("64Mi")` turns a Kubernetes quantity into a number, so `64Gi` and `64Mi`, or `500m` and `1`, can be compared (comparing the strings is wrong: "9Mi" sorts after "10Mi").

```yaml
schema: strict                         # strict (default) | off
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

Trade-offs considered: JSONPath is simpler to read but cannot express "all keys of A are in B"; Rego/OPA is a big dependency and a second language; Go test files per exercise would make contributors write Go and would put the answer in the learner's workspace. CEL is the middle: one small, well-known dependency (`cel.dev/cel-go`), declarative, safe (no side effects, bounded cost).

Rules:
- Messages are written for the learner and never contain the answer. Say what is wrong, not what to type.
- Every assertion is idempotent and pure. A manifest check that depends on the clock or the network is a bug.
- Do not assert on defaulted fields the API server would fill in; assert only what the learner wrote.
- Offline does not mean unvalidated: if a mistake would be rejected by the API server (wrong type, unknown field, missing required field), the schema step must catch it, and an exercise based on such a rejection uses `schema: strict` as the "check" with an empty `objects`.

### cluster — apply to a real cluster

The runner applies `exercise.yaml` to a fresh namespace in the k8slings cluster, waits for real conditions, asserts, and cleans up. Use this mode only when the thing being taught is runtime behaviour: rollouts, probes actually gating traffic, a Service getting endpoints, a Job completing, PVC binding, scheduling, NetworkPolicy blocking traffic, RBAC `can-i`, quota admission.

Flow: marker gate, cluster reachable, wipe and recreate the namespace `k8slings-<name>`, `kubectl apply -n <ns>`, then poll the check every 2 s until it passes or the timeout expires, then print diagnostics (pod states, events, last logs) if it did not. Exercise YAML never sets `metadata.namespace`.

`check.yaml` for cluster mode is declarative too, using the same CEL against **live** objects, with a small step vocabulary. The static part of the file (`schema`, `objects`) runs first, on the manifest alone, so a typo is reported in milliseconds without touching the cluster. (YAML reads an unquoted `off` as `false`; `schema: off` is accepted anyway.)

```yaml
steps:
  - rollout: deployment/web                          # generation seen, all replicas updated and available
  - ready: {pod: client}                             # or {selector: app=web, min: 2}
  - assert:                                          # any CEL over a live object (`object`) or a list (`objects`)
      resource: deployment/web                       # or: resources: pods, selector: app=web
      expr: "object.spec.strategy.type == 'RollingUpdate'"
      message: "Use a rolling update."
      now: true                                      # about the spec, not the cluster catching up: fail at once
  - endpoints: {service: web, min: 2, port: 80}      # ready EndpointSlice addresses
  - http: {service: web, port: 80, path: /, body_contains: nginx}
  - exec: {pod: intruder, args: [wget, -qO-, -T, "3", http://server], ok: false}
  - exec: {selector: app=client, args: [wget, -qO-, http://web], contains: nginx}   # a Deployment's Pods have no name to write: the first Running Pod that is not being deleted
  - can_i: {as: reader, verb: get, resource: pods/log, ok: true}   # a ServiceAccount of the exercise namespace; pods/log is the log subresource
    now: true                                       # `now` works on every step: an apply decides RBAC at once
  - logs: {pod: app, contains: "ready"}
```

Steps are evaluated in order and the first one that does not hold is reported; the runner polls the whole list every 2 s until it passes or the exercise's timeout (default 60 s) is up, then shows the problem and what the namespace looks like (Pods and the last warnings). A step marked `now: true` (any step type) ends the run immediately when it fails, because waiting cannot fix a mistake in what the learner wrote, or an answer an apply decides at once such as RBAC. Note that `kubectl auth can-i get pods/log` means "the Pod called log", so `can_i` passes a subresource as `--subresource`. An `exec` step first requires its Pod to be Running, so a Pod that has not started cannot count as "blocked". `http` goes through the API-server service proxy (`kubectl get --raw /api/v1/namespaces/<ns>/services/<name>:<port>/proxy/`), so no port-forward process, no host port, no firewall prompt, and it works the same on Colima, Lima and Docker Desktop. Traffic-policy questions (NetworkPolicy) must not use it, because the proxy bypasses pod-to-pod policy; those use `exec` from a client pod inside the namespace. Ingress/Gateway exercises are the one place that needs host reachability; see Add-ons.

Why declarative and not a Go `Check` func per exercise (as the first prototype did, with a `registry` of Go `Check` funcs): a declarative file is reviewable without reading Go, shares one evaluator and one set of timeouts, produces uniform failure output, and lets the same file drive fix mode. The cost is a fixed vocabulary. The planned escape hatch is a **named Go check** (`go: <name>` in `check.yaml`, resolved in a registry), for checks the vocabulary cannot express. It is **not built**: no exercise has needed it yet, and adding one needs a sentence in the PR on why the vocabulary was not enough. Named checks would follow the same purity rules.

A cluster exercise can also ship **cheats as files**: `solutions/<topic>/<name>/shortcut-*.yaml`, an exercise file whose first line is `# rejected: "text"`. `TestClusterSolutions` runs each one and requires the check to refuse it with that text **at once** (within seconds, by the static `objects:` part, before anything is applied), so a cheat never costs the learner a timeout. A guard written for this has to hold in the shipped state too, or it becomes the shipped reason and gives the fix away: state what must *stay as it is* (a container's name and command, the replica count), and let the runtime steps judge the fix.

All checks are **read-only and idempotent**: they are polled repeatedly. They must never mutate the cluster. A step that fails reports the first failing step with expected-versus-actual, not a dump.

### fix — troubleshooting with live state

CKA-style practice. The tool brings the cluster to a broken state; the learner repairs it with `kubectl` (through `k8slings kubectl`); the check looks at the result. There is **no file to edit**, so the mechanics differ from the other modes:

- **Files.** `exercises/<topic>/<name>/` has `README.md` (the ticket: the symptom, the rules, where to start; it is the only file `init` gives the learner), `setup.yaml` (the broken state, applied to the exercise's namespace; optionally followed by `setup-2.yaml`, `setup-3.yaml` ... up to five steps, see Seeding) and `check.yaml` (the steps, `shipped:`). The setup files and `check.yaml` stay in the binary because they would give the cause away; `exercises.EmbeddedOnly` is the one function that says which files those are, and init, reset and doctor all use it. `solutions/<topic>/<name>/` has `solution.sh` (a kubectl script; it calls plain `kubectl -n "$NS" ...`, and the test prelude makes that kubectl the k8slings one) and one or more `shortcut-*.sh`.
- **State.** Completion is kept in `.k8slings.json` next to `info.toml` (there is no marker line to delete), together with the UIDs of the seeded objects. It is written atomically, and a damaged file means "no progress", never a crash. **Done is sticky**: once the check has passed, the exercise stays done even if the learner breaks the cluster again later.
- **Seeding.** The first `run` or `watch` on a pending exercise seeds it: empty the namespace, label it `k8slings.io/exercise=<name>`, apply `setup.yaml`, record the UIDs of the objects the check names with `unchanged`. **Several steps** exist for exercises that need history (a working revision 1, then a broken revision 2 to `kubectl rollout undo`): the steps are applied in order, and every step except the last must finish rolling out first (the seeder runs `kubectl rollout status` for each Deployment, StatefulSet and DaemonSet the step applied, 90 s each), so only the last step may be broken, and it is not waited for. A step that does not roll out stops the seeding with a message that names the step. The namespace is labelled before step 1, and the UIDs are recorded only after the last step, so a failed or cancelled seeding leaves the label but no UIDs, and the next run seeds again from scratch (the old UIDs are dropped before a seeding starts). `run` and `watch` show "setting up <name>: step 1 of 2", and cancelling (`q`, ctrl+c) stops it with nothing recorded. Setup files are valid YAML on their own, never set `metadata.namespace`, and are validated by an offline test. It is reported as a hand-over ("the cluster is now broken on purpose"), not as a failure, and **nothing ever seeds again by itself**: that would wipe the learner's repair. A namespace that is labelled but that this workspace did not seed (another folder, or a lost progress file) is seeded fresh, because without the recorded UIDs it cannot be judged. `verify` and check-all run with `exercises.NoSeed(ctx)` and skip an exercise that was not started: checking everything must not break the cluster behind the learner's back.
- **Checking.** One look at the cluster, not a wait with a timeout: the learner is working in it. `watch` looks again every two seconds while the exercise is not repaired (it keeps going while the learner reads a hint), and turns green by itself; `run` checks once. **A repair must hold**: a check that passes is repeated after three seconds (`Env.Settle`) and only then counts, because a crash-looping container is briefly Running and Ready before it exits, and one look that lands there would call a broken cluster fixed (seen for real while building the first exercise).
- **`unchanged: deployment/web`** is a step: the live object must be the one that was seeded. Deleting a Deployment and creating a new one with the same name changes its UID and is refused. If the UIDs were not recorded, the step says so instead of passing.
- **`reset <name>`** breaks the cluster again (seed again) and forgets that it was done; the full-screen view asks first.
- **Shortcuts are proven refused.** Each `shortcut-*.sh` starts with `# rejected: "text"`, and `TestFixSolutions` seeds the exercise, runs the script, and requires the check to keep failing with that text, and the exercise not to be recorded as done. Every fix exercise needs at least one shortcut and at least one `unchanged` step (an offline test requires both). Other ways to cheat are blocked with ordinary steps marked `now: true` (keep the image, keep the replica count).
- **Safety.** A fix exercise only touches its own namespace. Cluster-scoped changes (a taint, a ClusterRole) and a `setup.sh` for things that are not plain manifests are **not built yet** (a script would break the single point where programs run and cannot be reviewed as YAML; multi-step YAML covers the need for history); exercises that need them wait for that, and a fix exercise that taints a node would need `requires = ["multinode"]` and a cleanup.

## Cluster handling and safety

Hard rule: **k8slings only ever talks to its own cluster, and never to the user's current context.**

- `kind create cluster` by default writes into `~/.kube/config` and switches `current-context`. That violates the rule, so `cluster up` runs `kind create cluster --name k8slings --kubeconfig <owned file>`. The owned file lives under the OS user config dir (`os.UserConfigDir()/k8slings/kubeconfig`). It is never merged into `~/.kube/config`.
- Every `kubectl` invocation passes `--kubeconfig <owned file> --context kind-k8slings` and runs with `KUBECONFIG` removed from its environment. There is one function that builds kubectl command lines; a unit test fails if any other code path execs `kubectl`.
- Before any cluster or fix exercise runs, the runner verifies that the API server in the owned kubeconfig is the one published by the `k8slings-control-plane` container (compare the server port with `docker port`). Any mismatch, a missing container, or a context name other than `kind-k8slings` is a hard refusal with an explanation. There is no flag to override it.
- Cluster-scoped changes are limited to the k8slings cluster by construction, but exercises still prefer namespaced objects. Cluster-scoped objects (ClusterRole, StorageClass, PriorityClass, taints) are labelled and removed by label.
- `cluster up` is idempotent and reuses an existing cluster. `cluster down` deletes only the cluster named `k8slings` and the owned kubeconfig.

### Version pinning

The target Kubernetes version is the newest that kind supports, **checked, not assumed**: kind v0.33.0 (released 2026-08-26) ships `kindest/node:v1.37.0@sha256:a1ed56cf…580ae5` as its default image; Kubernetes v1.38 is still alpha. `cluster up` passes the image **with its digest** (`--image`), so the cluster is identical on every machine. The full digest lives in one Go constant next to the minimum kind version. `doctor` warns when kind is older than the version the image was published with, and when `kubectl` is more than one minor version away from the cluster. Bumping Kubernetes is a deliberate PR: new node image, regenerated schemas, `k8s_version` reviewed, all exercises re-proven. A weekly job (`scripts/version-watch.sh`, `.github/workflows/version-watch.yml`) opens an issue when a newer kind release, or a newer default node image, is out (Dependabot cannot see constants in code).

### Topology

Node topology is fixed at cluster creation, so it is a Phase 0 decision rather than a per-exercise one: DaemonSet, taint, anti-affinity and spread exercises are meaningless on one node. Default: **one control-plane plus two workers** (about 1.2–1.5 GB of RAM idle, fine inside a 6 GB VM). `cluster up --single` creates one node for small machines; exercises with `requires = ["multinode"]` are skipped with a message. The create-time kind config is generated from the add-ons the user asked for, so enabling one that needs `extraPortMappings` means recreating the cluster, and `cluster up` says so instead of silently doing it.

### Container runtime

Docker on Linux is the baseline, and is what CI runs. On macOS, **Lima with the Docker template** (a Docker-compatible socket, or `DOCKER_HOST` pointing at it) is what the maintainer uses and what the real-cluster tests were run on. Docker Desktop and Colima (`colima start --vm-type vz --cpu 4 --memory 6`) are expected to behave the same, and `doctor` recognises them, but they are not tested. Lima with only containerd/nerdctl uses kind's experimental provider and is **best effort, not tested**. `doctor` reports which runtime and socket it found and prints the exact fix for the common failure (Docker not running, wrong `DOCKER_HOST`, the inotify limits below).

### Doctor

`k8slings doctor` reports two halves. **Offline mode** (the exercises folder, the embedded schema version) needs nothing but the binary, so its problems are failures. **Cluster mode** checks, in order: the container runtime (Docker Desktop, Colima, Lima, Rancher Desktop, Podman or Docker on Linux, with the Docker version); `kind` at or above the pinned minimum; `kubectl`; the Docker host's inotify limits (only when the node image is already on the machine, so that looking never starts a pull of about a gigabyte, with the fix for this runtime when they are too low); the cluster (exists, its kubeconfig leads to its own API server, every node Ready, CoreDNS, kube-proxy, kindnet and the storage provisioner Running, with the inotify explanation when kube-proxy crashed); and the Kubernetes version of the cluster against the pinned target and against kubectl's one-minor-version skew. A missing tool or a cluster that does not exist yet is only a **warning**, because manifest exercises do not need them; a cluster that exists but is broken is a **failure**. The exit code is non-zero only for failures, and the last lines say whether each half is ready.

## Add-ons *(target)*

Default kind has no metrics-server, no ingress or Gateway controller. **NetworkPolicy needs no add-on:** kind's default CNI (kindnet) has enforced NetworkPolicy via `kube-network-policies` since v0.24, and `networkpolicies1` tests real blocked traffic (confirmed on the pinned image: the client gets through, the intruder times out).

An exercise declares what it needs in `info.toml`, the way golings declares `go_version` and `race`:

```toml
requires = ["multinode", "metrics-server"]    # topology: multinode; add-ons: metrics-server, gateway
```

Implemented today: `requires` in `info.toml`, `multinode`, and the skip message. The add-on registry itself is Phase 5; it will have, per add-on: name, pinned manifest in `addons/`, a readiness check, and whether it needs create-time cluster config. `cluster up --addons metrics-server,gateway` installs them; `cluster addon install <name>` installs later when no recreate is needed. When `run` hits an exercise whose `requires` is not satisfied it skips with one line, like this one for add-ons: `needs metrics-server: run k8slings cluster addon install metrics-server`. `list` shows the lock, `check all` reports skipped, `verify` treats skipped as not-done-but-not-failed. Planned add-ons: `metrics-server` (HPA; needs `--kubelet-insecure-tls` on kind), `gateway` (Gateway API CRDs plus one implementation). **ingress-nginx is retired** (the Kubernetes project announced it in November 2025 and stopped maintenance in March 2026), so it is not an option; the chosen controller is Traefik (ROADMAP.md, "Decided"), pending a Phase 5 spike that it serves both Ingress and the Gateway API. Manifests are pinned to exact versions and live in the repo, not fetched from `main` at install time.

## Speed

Cluster exercises must check quickly. Rules: reuse one cluster; one namespace per exercise; poll every 2 s with a per-exercise timeout (`timeout` in `info.toml`, 30 to 120 s, default 60 s); empty the namespace rather than delete it (see "Lessons kept"); pre-load the handful of images exercises use (`cluster.PreloadImages`: `nginx:1.27`, `busybox:1.37`) into every node during `cluster up`, by pulling inside each node (`crictl pull`), so no exercise depends on a registry pull. `kind load docker-image` is deliberately not used: it fails ("content digest ... not found") when Docker uses its containerd image store. A test requires that every image a cluster exercise uses is in the list. An exercise whose solution takes more than about 30 s on CI hardware needs a reason. An unsolved exercise whose mistake only shows at runtime waits for its whole timeout; use `now: true` on steps about what the learner wrote so that those fail at once.

## CI

Workflows in `.github/workflows/`:

- **`test.yml` (every PR, no cluster):** formatting, `go vet`, all unit tests and `TestSolutions` for every **manifest** exercise (unsolved fails, solution passes; no solution without an exercise; no exercise without a solution), plus the manifest consistency tests and the docs-counts test. Go 1.26, 1.27 and stable on Linux, and the tests and the `go install` smoke test on Windows, since offline mode must work there.
- **`cluster.yml` (kind in GitHub Actions):** installs kind at the pinned minimum, raises the inotify limits, and runs `make cluster-test`: `TestRealCluster` (the safety rules, images on every node) and `TestClusterSolutions` (each cluster exercise must fail as shipped, for the reason its `shipped:` line names, and pass with its solution) and `TestFixSolutions` (each fix exercise: seed, fail for its reason, the solution script repairs it and it holds, every shortcut refused), all on one shared cluster per shard. The job is a matrix of four shards (`K8SLINGS_SHARD=i/4`: each shard proves every fourth cluster and fix exercise on its own cluster; shard 0 also proves the safety rules), because one cluster took more than the 30 minute limit once there were twelve troubleshooting exercises and a capstone (about 25 minutes locally, the job was cancelled at 30 on CI), and three shards had reached 14 to 15 minutes each once the content batches had added more cluster exercises (about 30 minutes of work in all). Without `K8SLINGS_SHARD`, `make cluster-test` runs everything on one cluster. It runs on every PR that touches exercises, solutions, the cluster, check, exercises or cmd packages, `go.mod` or the workflow itself; on every push to `main`; weekly; and on demand. Measured on GitHub's `ubuntu-latest`: about 4 minutes with two cluster exercises and **about 6 minutes with ten**, within the budget of about 10 minutes for running on every PR. Re-measure each shard as exercises are added; if a shard ever exceeds about 15 minutes, raise the shard count in `cluster.yml` (the `i/4` and the matrix), or move it to `main` plus weekly plus a PR label, and record that in ROADMAP.
- **`macos-smoke.yml`:** the `go install` smoke test on macOS, weekly and on demand, not on pull requests (GitHub's macOS runners are often unavailable, and a PR must not wait for them).
- **`release.yml`:** on a `v*` tag, GoReleaser builds the archives and the release body is the CHANGELOG section (see CONTRIBUTING.md, "Releasing").
- **`version-watch.yml`:** weekly, opens an issue for a newer kind or Kubernetes node image.
- **Add-on job (Phase 5):** exercises that need metrics-server or the gateway controller will run in a separate job, so that a flaky add-on does not block the basic cluster exercises.
- A consistency test keeps docs honest: the README carries markers like `<!-- count:exercises -->`, and a test compares them with `info.toml`. **Do not hardcode exercise counts in this file** (or anywhere outside the marked places).

## Rules for exercises (all modes)

- Each exercise is small and single-concept. A `README.md` says what to learn in a few lines; the YAML has `# I AM NOT DONE` on line 1 and a `# TODO:` comment at the fix site, one sentence, without giving the answer away (the solution does not keep either).
- Idiomatic, current Kubernetes only: `apps/v1`, `networking.k8s.io/v1`, no removed APIs, no `kubectl run --generator`, no Endpoints where EndpointSlices are meant, pinned image tags (never `latest`), `securityContext` shown in the secure style where it is not the topic.
- The unsolved version must fail every time, the same way. Failures should come from the exercise, not from a broken cluster.
- Never weaken a check just to make a solution pass. If bug, solution and check disagree, fix the exercise.
- Every cluster exercise declares `shipped: "..."` in `check.yaml`: what the first problem of the unsolved exercise says. `TestClusterSolutions` requires the unsolved exercise to fail with that, and an offline test requires the line. A bare "it failed" proves little (a slow cluster fails everything), and an exercise whose premise is false passes its own checks: `statefulsets2` once assumed a normal Service has no per-Pod DNS, which is wrong on this cluster, and it was only caught because CI's machine was faster than the 15 s proof timeout. Use fully qualified names in checks, so that a short name cannot fall back to public DNS. When the first problem is generic ("0 of 1 replicas are available"), add `shipped_details: "text"`: the proofs then also require the diagnosis of the namespace (Pods, warnings) to show it, which proves the cause (`OOMKilled`, a missing StorageClass, a node affinity that matches nothing). It is only for the proofs and never changes whether an exercise passes; keep it a short fragment, because Kubernetes writes event wording and a version bump re-proves it.
- "Fails as shipped, passes solved" is not enough proof. Also try, on a real cluster for cluster exercises, each fault fixed on its own, the shortcuts (deleting the thing being tested, raising the quota, allowing everyone), and a legitimate alternative answer, and check that each gets the right result and a message that helps. This found a check that never verified pod logs, and a diagnosis full of stale warnings.
- Every exercise has a verified solution in `solutions/`, mirroring `exercises/`. Order matters: put new exercises where they belong in the learning progression.
- Pick the cheapest mode that can decide correctness. Promote manifest to cluster only if runtime behaviour is the lesson.
- Cluster and fix exercise YAML never sets `metadata.namespace` (k8slings picks it). Manifest exercises may, when the namespace is the lesson.
- No behaviour that depends on a registry rate limit, the wall clock or the order of API list results.
- Windows: manifest mode and all paths in `info.toml` use forward slashes; no file permissions in tests.

## Commands for contributors

```sh
make test            # unit tests and the offline TestSolutions (no cluster needed)
make cluster-test    # needs docker + kind + kubectl; creates real clusters and proves the safety rules and every cluster exercise (about 6 minutes)
make smoke           # installs into a temp dir, runs init, list, doctor, run on the first exercise
make watch / make run
```

## Lessons kept from the prototype

The first version of k8slings (a cluster-only prototype with 19 exercises, removed from the tree once every exercise was migrated; see the git history) taught things that are still true:

- A namespace is emptied in a fixed order: the controllers first (Deployments, StatefulSets, DaemonSets, ReplicaSets, Jobs, CronJobs) with `--cascade=orphan`, then every other namespaced resource type from `kubectl api-resources` (no kind list to maintain) with `--now`, then we wait until no Pod is left, and for `serviceaccount/default` to reappear, because Pods are rejected until it exists. The order matters: deleting Pods in the same command as their Deployment lets the ReplicaSet create new Pods meanwhile, which the garbage collector removes later, after the reset has returned, and they count against the next run's ResourceQuota ("exceeded quota ... used: pods=4" with two Pods in sight). That made a Deployment stay one Pod short for the whole timeout, 2 times in 45 on GitHub's runners and never on a laptop. Foreground cascading looks like the answer but is not: the garbage collector then uses the 30 s default grace period, and a Pod whose process ignores SIGTERM (busybox `sleep`) makes every run wait that long. Events are cleared too, or the diagnosis of a failed run shows the warnings of earlier runs.
- **A finalizer that nothing handles** (an exercise leaves one on an object on purpose: troubleshooting15) blocks emptying the namespace and deleting it, for ever; the delete used to hang for its whole 60 s and fail. `ResetNamespace`, and the proofs before they delete a namespace, call `ReleaseFinalizers`: in a namespace named `k8slings-*`, and only there, it removes the finalizers that are not the cluster's own (`kubernetes.io/*`, `foregroundDeletion`, `orphan`). A finalizer has to be fully qualified (`example.com/cleanup`) or the API server rejects the object.
- Some mistakes are rejected by the API server's *semantic* validation, not by the OpenAPI schema (a Deployment selector that does not match its template labels, a Job with `restartPolicy: Always`). In manifest mode they must be reproduced with a CEL assertion; in cluster mode the rejected apply is reported as the mistake.
- `kubectl auth can-i get pods/log` asks about the Pod called `log`, not the log subresource. The prototype's RBAC check never verified pod logs because of it; `can_i` now passes `--subresource`.
- The scheduling exercise cannot target the control-plane node: in a multi-node cluster it keeps its `NoSchedule` taint.
- Reading EndpointSlices with `?(@.conditions.ready==true)` depends on the node image, so re-prove the services exercises on every Kubernetes bump.
- **Docker VM inotify limits** (Colima/Lima): a 3-node cluster needs `fs.inotify.max_user_instances` of at least 512 (stock Lima has 128), or `kind create` reports success while the workers never join and kube-proxy crashes with "too many open files". `cluster up` and `doctor` check this and print the fix for the runtime.

## Licensing and attribution

Code copied or adapted from golings stays under golings' **Apache-2.0** license, with a `NOTICE` that credits both golings (Cristian Leonte) and the original mauricioabreu/golings, and `LICENSE` kept intact. k8slings' own new code and exercise content are also **Apache-2.0** (decided 2026-11-04), so the repository has a single license. `LICENSE` and `NOTICE` are in the repository root. Text and manifests adapted from `../learn/k8s-2-weeks` (MIT code, CC BY 4.0 text) need attribution in `NOTICE`.

## Working style

- Work on a branch per roadmap step, never straight on `main`, and open a PR per step. Branch names like `phase-1-skeleton`.
- Small commits, one logical change each. Run `make test` before every commit; for anything touching cluster, add or run `make cluster-test` for the exercises affected.
- Tests stay green on every commit. A new exercise without a solution fails the build on purpose.
- Do not rewrite an exercise's intent when modernising; fix idioms and wording only.
- Commit messages and PR descriptions carry no `Co-Authored-By` or "Generated with" trailers.
- When unsure about a format, read the golings code (`../golings/golings/exercises/`), then an existing k8slings example, then follow them.
- Changes to this file and ROADMAP.md that alter a decision go in their own commit, with the reason.
