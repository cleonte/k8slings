# deployments2: updating without downtime

When you change a Deployment's Pod template, its **strategy** decides how the old Pods are replaced:

- `Recreate` stops every old Pod first, then starts the new ones: a window with no Pods at all.
- `RollingUpdate` (the default) replaces them gradually. `maxUnavailable` is how many Pods may be missing during the update, and `maxSurge` how many extra Pods may exist.

This exercise runs on the cluster: the Deployment is applied and k8slings waits for it to roll out. Make updates zero-downtime: never fewer Pods than wanted (`maxUnavailable: 0`), at most one extra (`maxSurge: 1`).
