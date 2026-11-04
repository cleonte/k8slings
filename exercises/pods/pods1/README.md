# pods1: your first Pod

A **Pod** is the smallest thing Kubernetes runs: one or more containers that share a network and storage.

Every Kubernetes object has `apiVersion`, `kind`, `metadata` and (for most kinds) a `spec`. For a Pod, `spec.containers` is a list, and each container needs a `name` and an `image`.

The file you have is rejected by Kubernetes. Run `k8slings run pods1` to see why, fix it, then remove the `I AM NOT DONE` line.
