# Security policy

## Reporting a vulnerability

Please report security problems **privately**, not in a public issue: open the repository's **Security** tab and choose **Report a vulnerability** (GitHub's private vulnerability reporting). Include what you did, what you expected, and what happened, and the output of `k8slings --version`.

k8slings is a small, volunteer-maintained project. I will read every report and answer as soon as I can, but I cannot promise a response time.

## What counts

k8slings runs `docker`, `kind` and `kubectl` on your machine and creates a local kind cluster, so the things that matter most are the promises it makes about that:

- **It must only ever talk to its own cluster** (`kind-k8slings`, with its own kubeconfig file), never to your current context or any other cluster. Anything that makes it read, write or send commands to another cluster is a vulnerability.
- It must not change your own `~/.kube/config` or your current context.
- It must not run commands you did not ask for, or leak your kubeconfig or credentials.
- The release archives must be what the release workflow built from the tagged source.

Problems with the exercises themselves (a wrong answer accepted, an unclear lesson) are not security issues: use the normal issue templates.

## Supported versions

Only the latest release is supported. Fixes go into a new release.
