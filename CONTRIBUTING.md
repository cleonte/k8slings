# Contributing to k8slings

Thanks for helping. The design is in [CLAUDE.md](CLAUDE.md) and the plan in [ROADMAP.md](ROADMAP.md); read the exercise rules in CLAUDE.md before you add one.

## Adding a manifest exercise

An exercise is a folder `exercises/<topic>/<name>/` with three files, plus a solution and a manifest entry:

1. `README.md`: the lesson, a few lines.
2. `exercise.yaml`: the broken manifest. Line 1 is `# I AM NOT DONE`. A `# TODO:` comment sits at the fix site and says what to do in one sentence, without giving the answer. Do not set `metadata.namespace` in cluster and fix exercises (k8slings picks the namespace); manifest exercises may, when the namespace is the lesson.
3. `check.yaml`: the schema mode and the CEL assertions. Every assertion has a `message` written for the learner. Quote an `expr` that contains `: ` (a map literal such as `{'app': 'web'}`), or YAML will misread it. This file is embedded in the binary and is not written to the learner's folder.
4. `solutions/<topic>/<name>/exercise.yaml`: a working version, without the marker and the TODO.
5. An entry in `info.toml`, **in learning order** (not at the end by habit):

```toml
[[exercises]]
name = "pods1"
path = "exercises/pods/pods1"
topic = "pods"
mode = "manifest"
hint = "A short nudge."
docs = "https://kubernetes.io/docs/concepts/workloads/pods/"
```

If the exercise uses a kind that has no embedded schema, add it to `scripts/gen-schemas.sh` and run the script.

`go test ./...` checks all of it: the layout, the marker and TODO, the check compiles, the unsolved version fails with a message, and the solution passes.

## Tests

```sh
go test ./...        # everything that needs no cluster
make smoke           # go install into a temp folder, then run what a learner types
                     # (macOS runs it weekly in CI, macos-smoke.yml, not on pull requests)
```

Tests must pass on Windows too: no `/tmp`, forward slashes in `info.toml`, no file permissions.

## Regenerating the demo GIFs

The GIFs in the README are recorded from scripts with [vhs](https://github.com/charmbracelet/vhs) (`brew install vhs`, which also installs `ttyd`). Install the k8slings of your checkout first, then run a tape:

```sh
go install ./k8slings
vhs demo-tui.tape   # misc/demo-tui.gif: the full-screen `k8slings watch`
vhs demo.tape       # misc/demo.gif: `k8slings run`
```

## Updating the Kubernetes schemas

`scripts/gen-schemas.sh [version]` regenerates `schemas/`. Bumping the Kubernetes version is a deliberate change: it is a separate PR and every exercise is re-proven.

## Releasing

Not set up yet (Phase 6 in the roadmap).
