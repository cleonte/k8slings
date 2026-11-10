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

## Adding a cluster exercise

Use `mode = "cluster"` only when the lesson is runtime behaviour (a rollout, endpoints, traffic). `check.yaml` then has `steps`, see [CLAUDE.md](CLAUDE.md) for the vocabulary. Also:

- Do not set `metadata.namespace`: k8slings applies the manifest to its own namespace for the exercise.
- Use `now: true` on a step that is about what the learner wrote (a field of the spec) or that an apply decides at once (RBAC with `can_i`). Without it, a mistake that waiting cannot fix is reported only after the timeout.
- Set a `timeout` that the solution passes in with room to spare; remember that an unsolved exercise waits for all of it unless a `now` assertion fails first.
- Images must be listed in `cluster.PreloadImages` (a test checks it), so that no exercise depends on a registry.
- Say why the unsolved exercise fails: `shipped: "text"` at the top of `check.yaml` is a part of the first problem it reports, and `make cluster-test` requires it. Failing is not enough, because a slow cluster fails every exercise; one exercise passed its own checks on CI because its premise was wrong, and only this line makes that visible.
- In a check, use fully qualified names (`web-0.web.k8slings-<name>.svc.cluster.local`), not short ones: a short name falls back to public DNS after the cluster's search domains.
- Prove it with `make cluster-test` (needs Docker, kind and kubectl): the unsolved exercise must fail for the reason in `shipped`, and the solution must pass. Then run the unsolved exercise once with the real CLI and its real timeout and read why it fails.

## Adding a fix exercise

A fix exercise has no file to edit: k8slings breaks the cluster, and the learner repairs it with `kubectl`. Use it for troubleshooting, where the skill is to look and diagnose. In `exercises/<topic>/<name>/`:

- `README.md`: the ticket (the symptom, the rules, where to start). It is the only file the learner is given.
- `setup.yaml`: the broken state. Do not set `metadata.namespace`. It stays in the binary, because it would give the cause away.
- `check.yaml`: `shipped: "..."` (what the first problem says), optionally `shipped_details: "..."` (a fragment the diagnosis must show, when the first problem alone does not tell the cause), and `steps`. It must have an `unchanged: kind/name` step, so that deleting the object and creating a new one does not count as a repair, and `now: true` assertions for the other ways to cheat (keep the image, keep the replicas).
- `solutions/<topic>/<name>/solution.sh`: a kubectl script. Write plain `kubectl -n "$NS" ...`; the tests make that kubectl the k8slings one.
- `solutions/<topic>/<name>/shortcut-*.sh`: at least one way to cheat. The first line is `# rejected: "text the check must say"`. `make cluster-test` runs it and requires the check to keep failing with that text.

A fix exercise that has a crashing or flapping Pod needs nothing special: a repair has to hold for three seconds before it counts. Images must be in `cluster.PreloadImages`, as for cluster exercises.

## Tests

```sh
go test ./...        # everything that needs no cluster
make cluster-test    # needs Docker, kind and kubectl; creates real clusters, checks the safety rules, proves every cluster and fix exercise; can take up to 40 minutes
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

Pushing a tag `vX.Y.Z` runs the release workflow, which builds the archives (Linux, macOS and Windows, on amd64 and arm64) with [GoReleaser](https://goreleaser.com) and publishes the release. The body of the release is the section of `CHANGELOG.md` for that version (`scripts/release-notes.sh vX.Y.Z` prints it, and fails when there is none, in which case GoReleaser writes the notes from the commits).

So before tagging:

1. Move what is under "Unreleased" in `CHANGELOG.md` into a section `## [X.Y.Z] - date`, and update the two links at the bottom.
2. Check the release locally: `goreleaser check`, then `goreleaser release --snapshot --clean --skip=publish` builds every archive into `dist/` without publishing anything.
3. Tag and push: `git tag vX.Y.Z && git push origin vX.Y.Z`.

The release also updates the Homebrew cask in the tap [cleonte/homebrew-tap](https://github.com/cleonte/homebrew-tap) (`homebrew_casks` in `.goreleaser.yaml`). That needs the repository secret `HOMEBREW_TAP_TOKEN`: a fine-grained personal access token with **Contents: read and write on the tap repository only** (the `GITHUB_TOKEN` of a workflow cannot push to another repository). Without it the release fails at the publish step, after the archives are built. The token has an expiry: when it lapses, create a new one and `gh secret set HOMEBREW_TAP_TOKEN`.
