# rbac2: write the policy yourself

This time nothing is fixed for you: you write the Role and the RoleBinding. Things to watch:

- The `apiGroups` of a resource: core resources such as Pods use `""`, Deployments use `apps`. `kubectl api-resources` shows the group of every resource.
- Subresources such as `pods/log` are separate resources with their own rule.
- A Role only grants access inside its own namespace.
- **Least privilege**: grant exactly what is listed, no wildcards.

The ServiceAccount `deployer` must be able to:

- `get`, `list` and `watch` Pods
- `get` the logs of Pods (`pods/log`)
- `get`, `list` and `patch` Deployments (group `apps`)

and must **not** be able to `create` or `delete` Deployments, `delete` Pods, read Secrets, or list Pods in another namespace (say `kube-system`). This exercise runs on the cluster, and k8slings asks the API server about every one of these.
