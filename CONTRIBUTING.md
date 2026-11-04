# Contributing to k8slings

Thanks for helping. The design is in [CLAUDE.md](CLAUDE.md) and the plan in [ROADMAP.md](ROADMAP.md); read the exercise rules in CLAUDE.md before you add one.

## Adding a manifest exercise

An exercise is a folder `exercises/<topic>/<name>/` with three files, plus a solution and a manifest entry:

1. `README.md`: the lesson, a few lines.
2. `exercise.yaml`: the broken manifest. Line 1 is `# I AM NOT DONE`. A `# TODO:` comment sits at the fix site and says what to do in one sentence, without giving the answer. Never set `metadata.namespace`.
3. `check.yaml`: the schema mode and the CEL assertions. Every assertion has a `message` written for the learner. This file is embedded in the binary and is not written to the learner's folder.
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
```

Tests must pass on Windows too: no `/tmp`, forward slashes in `info.toml`, no file permissions.

## Updating the Kubernetes schemas

`scripts/gen-schemas.sh [version]` regenerates `schemas/`. Bumping the Kubernetes version is a deliberate change: it is a separate PR and every exercise is re-proven.

## Releasing

Not set up yet (Phase 6 in the roadmap).
