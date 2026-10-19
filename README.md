# k8slings

Small, broken Kubernetes manifests to fix, in the spirit of [Rustlings](https://rustlings.rust-lang.org/).
Runs on a local [kind](https://kind.sigs.k8s.io/) cluster named `k8slings`, so cleanup is one command.

## Requirements

`docker` (e.g. Colima or Docker Desktop), `kind`, `kubectl`, and Go 1.22+ to build.

## Quick start

```bash
go build -o k8slings .
./k8slings init            # creates ./k8slings-exercises and the kind cluster
cd k8slings-exercises
../k8slings watch          # runs the first exercise, re-checks each time you save
```

In watch mode: `h` + Enter = hint, `l` = list, `r` = re-run, `q` = quit.
Each exercise lives in `exercises/<id>/` with a `README.md` (the lesson) and `exercise.yaml` (what you edit).
Delete the `# I AM NOT DONE` line when you think you've fixed it.

## Commands

`init [dir]` · `watch` · `run [id]` · `list` · `hint [id]` · `reset [id]` · `verify` · `destroy`

`destroy` deletes the whole kind cluster; nothing else on your machine is touched.
All kubectl calls are pinned to the `kind-k8slings` context, so your other clusters are never modified.

## Design

- No dependencies beyond the Go stdlib; shells out to `kind` and `kubectl`.
- Exercises are embedded in the binary (`exercises/`); `init` extracts them (never overwrites your work).
- Each exercise gets its own namespace (`k8slings-<id>`), wiped before every check.
- A check = `kubectl apply` of your file, then poll a Go function (`exercises.go`) until it passes or times out.
- Progress is stored in `.k8slings.json` in the workspace.

## Adding an exercise

1. Create `exercises/NN_name/{README.md,exercise.yaml,hint.md}` (start `exercise.yaml` with the `# I AM NOT DONE` line).
2. Add its solution as `solutions/NN_name.yaml`.
3. Register it in `exercises.go` with a `Check` function.
4. `go build -o bin/k8slings . && bash scripts/e2e.sh` proves it fails when broken and passes when solved (needs docker, kind, kubectl).

## Roadmap

Exercises from the rest of the learn path (storage, scheduling, NetworkPolicy, Helm/Kustomize, troubleshooting drills with a pre-broken live cluster), a bubbletea list view, and Homebrew packaging.
