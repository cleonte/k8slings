# k8slings

Rustlings-style Kubernetes learning app. Go CLI, stdlib only, shells out to `kind` and `kubectl`.
Learner edits broken manifests; the app applies them to a local kind cluster and checks cluster state.
Source material for new exercises: `../learn/k8s-2-weeks/` (day-01 to day-10, extras/, core-advanced/, ckad-gaps/).

## Prerequisites

`docker` (or colima: `colima start --vm-type vz --cpu 4 --memory 6`), `kind`, `kubectl`, `go`.

## Commands

```bash
go build -o bin/k8slings . && go vet ./...
bash scripts/e2e.sh                      # full e2e, parallel (JOBS=4), ~2.5 min; pass ids for a subset; KEEP=1 skips destroy
./bin/k8slings init /tmp/ws && cd /tmp/ws && /path/to/bin/k8slings watch
./bin/k8slings run 01                    # single exercise (zero-padded or plain number)
```

CLI: `init watch run list hint reset verify destroy`. In `watch`: `h` hint, `l` list, `r` re-run, `q` quit (each + Enter).
Set `NO_COLOR=1` to disable ANSI colour/clear-screen.

`scripts/e2e.sh` uses a temp workspace and, per exercise, asserts: (1) untouched run is blocked by the marker,
(2) marker removed but bug still in: check FAILS, (3) `solutions/<id>.yaml` applied: check PASSES. Ends with `k8slings destroy`.
The "still broken" run uses `K8SLINGS_TIMEOUT=$BROKEN_TIMEOUT` (default 20s) instead of the exercise `Timeout`; the solution run uses the real one.
If an exercise's bug, solution and check disagree, fix the exercise. Never weaken a check just to make it pass.

There are no Go unit tests; e2e is the test suite.

## Layout

| Path | Role |
|---|---|
| `main.go` | command dispatch |
| `exercises.go` | `registry`: ID, title, timeout, `Check(*Env)`; `Env` helpers (`get`, `expect`, `logs`, `podRunning`) |
| `runner.go` | `Run`: marker gate, cluster check, wipe namespace, `kubectl apply -n <ns>`, poll `Check` every 2s until timeout, print diagnostics |
| `watch.go` | watch loop (mtime polling every 500ms, stdin commands), list output |
| `workspace.go` | `.k8slings.json` progress file, embedded exercise extraction (`//go:embed all:exercises`), never overwrites on re-init |
| `cluster.go` | kind create/delete, `kubectl()` wrapper pinned to context `kind-k8slings` |
| `ui.go` | ANSI colour helpers |
| `exercises/<id>/` | `README.md` (lesson), `exercise.yaml` (broken, first line `# I AM NOT DONE`), `hint.md` |
| `solutions/<id>.yaml` | known-good fix, used only by e2e |

19 exercises: 01-11 single-fault basics, 12-16 gap fillers (Secrets, PVC, init container, scheduling, NetworkPolicy),
17-19 harder (write RBAC from scratch, quota/LimitRange arithmetic, multi-fault capstone). Details: see `registry` in `exercises.go` and each `exercises/<id>/README.md`.

## Design rules

- Cluster `k8slings`, context `kind-k8slings`. Every kubectl call passes `--context`; never touch the user's other clusters.
- One namespace per exercise: `k8slings-<id with _ -> ->`. Exercise YAML must NOT set `metadata.namespace` (runner uses `-n`).
- Before each check, every namespaced resource type (from `kubectl api-resources`) is deleted, then `run` waits for `sa/default` to reappear. No kind list to maintain.
- Checks must be idempotent and read-only; they are polled repeatedly.
- No third-party Go deps unless there is a strong reason (a bubbletea TUI would be one).

## Adding an exercise

1. `exercises/NN_name/{README.md,exercise.yaml,hint.md}`; first line of `exercise.yaml` is the `# I AM NOT DONE` marker.
2. `solutions/NN_name.yaml`.
3. Append to `registry` in `exercises.go` with a `Check`.
4. Run `bash scripts/e2e.sh`.

## Fragile spots

- Docker VM inotify limits: if kube-proxy/coredns/local-path-provisioner crash with "too many open files", run
  `colima ssh -- sudo sysctl -w fs.inotify.max_user_instances=512 fs.inotify.max_user_watches=524288` and restart those pods.
  Symptoms in e2e: DNS "bad address" (16, 19), PVC stuck Pending (13).

- `Check` functions in `exercises.go` and namespace cleanup in `runner.go` are the likeliest sources of bugs.
- 03 and 10: the bug is an apply-time rejection; `Run` returns the apply error, so the e2e "still broken" case fails fast.
- 06 reads EndpointSlices via jsonpath filter `?(@.conditions.ready==true)`; depends on the kind node image.
- 01 and 08 pull nginx; first run is slow, timeouts may need tuning.
- 02/07 read pod logs right after Running; a race is possible, polling should absorb it.

## Roadmap

1. More exercises: storage/PVC (extras/02), scheduling and taints (extras/03), NetworkPolicy (extras/06),
   Secrets, init/multi-container pods (ckad-gaps/lab-11), HPA with metrics-server, Ingress, Kustomize/Helm.
2. "Drill" exercises: pre-broken live workloads fixed with kubectl (CrashLoop, OOM, bad selector), checked on end state.
3. `solution <id>` command, `--chapter` grouping, bubbletea list view.
4. GitHub Actions running e2e on a kind runner; release binaries and a Homebrew tap.
