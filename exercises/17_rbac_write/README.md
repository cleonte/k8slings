# 17 Write an RBAC policy

This time nothing is fixed for you: you write the Role and RoleBinding yourself. Mind the API group of each resource (`kubectl api-resources` shows it), subresources like `pods/log` are separate resources, and a Role only grants access inside its own namespace.

## Goal
ServiceAccount `deployer`, in its own namespace, must be able to:
- `get`, `list`, `watch` pods
- `get` the logs of pods (`pods/log`)
- `get`, `list`, `patch` deployments (API group `apps`)

and must NOT be able to:
- `create` or `delete` deployments
- `delete` pods
- read any secrets
- list pods in any other namespace (e.g. `kube-system`)

## Task
Add a Role and a RoleBinding to `exercise.yaml`. Least privilege: grant only what is listed.
