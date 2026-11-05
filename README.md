# k8slings

Learn Kubernetes by fixing small broken manifests, in the spirit of [rustlings](https://github.com/rust-lang/rustlings) and [golings](https://github.com/cleonte/golings).

> **Early development.** There are <!-- count:exercises -->15<!-- /count --> exercises in <!-- count:topics -->7<!-- /count --> topics so far, all of them *manifest* exercises that run offline. Cluster and troubleshooting exercises are on the [roadmap](ROADMAP.md).

![k8slings watch](misc/demo-tui.gif)

## How it works

You get a folder of Kubernetes manifests, each one broken in one small way. Edit a manifest until `k8slings` is happy, then delete the `# I AM NOT DONE` line at the top and move on.

There are three kinds of exercise (see [CLAUDE.md](CLAUDE.md) for the details):

- **manifest**: offline, no cluster. Your YAML is validated against the Kubernetes schema and checked with a few assertions. You can start with nothing but this binary.
- **cluster** *(planned)*: your manifest is applied to a local [kind](https://kind.sigs.k8s.io/) cluster and checked against what really happens.
- **fix** *(planned)*: the tool breaks something in the cluster, you repair it with `kubectl`.

## Install

You need Go 1.26 or newer:

```sh
go install github.com/cleonte/k8slings/k8slings@latest
```

## Doing exercises

```sh
k8slings init            # creates k8slings-exercises/
cd k8slings-exercises
k8slings watch           # check the first exercise that is not done yet, and again on every save
```

| Command | What it does |
|---|---|
| `k8slings watch` | Full-screen view that re-checks the current exercise every time you save. `--plain` for plain lines. |
| `k8slings run next` / `run <name>` | Check one exercise. |
| `k8slings hint <name>` | Show the hint and a link to the Kubernetes docs. |
| `k8slings cluster up` / `down` / `status` | Create, inspect and delete the local kind cluster (needs Docker, kind and kubectl; manifest exercises do not). |
| `k8slings kubectl ...` | kubectl against the k8slings cluster only. |
| `k8slings doctor` | Check the workspace and what the binary brings with it. |
| `k8slings list` | All exercises, by topic, with your progress. |
| `k8slings verify` | Check every exercise in order. |
| `k8slings solution <name>` | Show the solution (hidden until you solve it; `--reveal` to look anyway). |
| `k8slings reset <name>` | Restore an exercise to its original, broken state. |

In `watch`: `h` hint, `s` solution, `r` reset, `n` next, `l` list, `c` check all, `q` quit.

![k8slings run](misc/demo.gif)

Each exercise folder has a `README.md` with the lesson and an `exercise.yaml` to edit. Look for the `# TODO:` comment.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). New exercises need a solution, and CI proves that the broken version fails and the solution passes.

## License and credits

Apache License 2.0, see [LICENSE](LICENSE) and [NOTICE](NOTICE). The structure of this project comes from [golings](https://github.com/cleonte/golings), a fork of [mauricioabreu/golings](https://github.com/mauricioabreu/golings).
