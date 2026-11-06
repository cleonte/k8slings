# k8slings

Learn Kubernetes by fixing small broken manifests, in the spirit of [rustlings](https://github.com/rust-lang/rustlings) and [golings](https://github.com/cleonte/golings).

You get <!-- count:exercises -->35<!-- /count --> exercises in <!-- count:topics -->17<!-- /count --> topics. Most of them need nothing but this program: your YAML is checked offline. Some run against a local cluster that k8slings creates for you, and check what really happens.

![k8slings watch](misc/demo-tui.gif)

## How it works

Each exercise is a Kubernetes manifest that is broken in one small way (the last one, in four). Edit it until `k8slings` is happy, then delete the `# I AM NOT DONE` line at the top and move on. Every exercise has a short lesson, a hint, and a link to the Kubernetes documentation.

- **Manifest exercises** run offline. Your YAML is validated against the Kubernetes 1.37 schema (a misspelled `contianers:` is caught, with a message you can read) and checked with a few assertions about what you wrote.
- **Cluster exercises** are applied to a local [kind](https://kind.sigs.k8s.io/) cluster of their own and checked against what really happens: a rollout completes, a Service has endpoints, a request gets through, a network policy keeps an intruder out, a ServiceAccount may or may not delete Pods. When it does not work yet, you see the Pods and the warnings of the cluster. Without the cluster, these exercises are skipped and the rest carry on.

What you will practise: Pods, labels and selectors, namespaces, Deployments and rollouts, Services, ConfigMaps and Secrets, probes, requests, limits and quotas, Jobs, volumes and claims, scheduling, RBAC, network policies, and a capstone: a broken stack to debug like a real incident.

## Install

Download the archive for your system from the [releases page](https://github.com/cleonte/k8slings/releases) and put `k8slings` on your PATH, or, with Go 1.26 or newer:

```sh
go install github.com/cleonte/k8slings/k8slings@latest
```

**What you need:**

| For | You need |
|---|---|
| Manifest exercises | Nothing else. |
| Cluster exercises | Docker (Docker Desktop, Colima, Lima with the Docker template, or Docker on Linux), [kind](https://kind.sigs.k8s.io/) v0.33.0 or newer, and `kubectl`. |

`k8slings doctor` checks all of it, says whether each half is ready, and prints the exact fix for anything that is not. One thing to know about Colima and Lima: their VMs allow too few inotify instances for a 3-node cluster by default, and `k8slings cluster up` tells you the one command that fixes it.

**On macOS**, the binaries are not signed or notarized. If you download an archive in a browser, macOS may refuse the first run ("cannot be opened because the developer cannot be verified"). Either download it with `curl`, install with `go install`, or remove the quarantine flag once: `xattr -d com.apple.quarantine k8slings`.

## Doing exercises

```sh
k8slings init            # creates k8slings-exercises/
cd k8slings-exercises
k8slings watch           # checks the first exercise that is not done yet, and again on every save
```

For the cluster exercises, create the cluster once (about two minutes the first time, and about 1.5 GB of memory):

```sh
k8slings cluster up      # one control plane and two workers; --single for one node
k8slings cluster down    # when you are done
```

The cluster is k8slings's own: it has its own kubeconfig, so your `~/.kube/config` and your current context are never touched, and k8slings never talks to any other cluster. To look around in it yourself, use `k8slings kubectl ...`, for example `k8slings kubectl get pods -A`.

| Command | What it does |
|---|---|
| `k8slings watch` | Full-screen view that re-checks the current exercise every time you save. `--plain` for plain lines. |
| `k8slings run next` / `run <name>` | Check one exercise. |
| `k8slings hint <name>` | Show the hint and a link to the Kubernetes docs. |
| `k8slings list` | All exercises, by topic, with your progress. |
| `k8slings verify` | Check every exercise in order. |
| `k8slings solution <name>` | Show the solution (hidden until you solve it; `--reveal` to look anyway). |
| `k8slings reset <name>` | Restore an exercise to its original, broken state. |
| `k8slings cluster up` / `down` / `status` | Create, inspect and delete the local kind cluster. |
| `k8slings kubectl ...` | kubectl against the k8slings cluster only. |
| `k8slings doctor` | Check the workspace, Docker, kind, kubectl and the cluster. |

In `watch`: `h` hint, `s` solution, `r` reset, `n` next, `l` list, `c` check all, `q` quit.

![k8slings run](misc/demo.gif)

Each exercise folder has a `README.md` with the lesson and an `exercise.yaml` to edit. Look for the `# TODO:` comment.

## Status

k8slings is young. The offline and cluster exercises described above work and are proven in CI on every change, including on a real kind cluster. Still to come, see the [roadmap](ROADMAP.md): troubleshooting exercises where you repair a broken live cluster with `kubectl`, more topics (StatefulSets, DaemonSets, security contexts, Kustomize), and exercises that need add-ons (HPA, Ingress and the Gateway API).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). New exercises need a solution, and CI proves that the broken version fails and the solution passes.

## License and credits

Apache License 2.0, see [LICENSE](LICENSE) and [NOTICE](NOTICE). The structure of this project comes from [golings](https://github.com/cleonte/golings), a fork of [mauricioabreu/golings](https://github.com/mauricioabreu/golings).
