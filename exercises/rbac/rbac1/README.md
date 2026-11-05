# rbac1: who can do what

**RBAC** decides what an identity may do. A **Role** lists permissions: which verbs (`get`, `list`, `delete`, ...) on which resources, in its own namespace. A **RoleBinding** gives a Role to a *subject*, such as a ServiceAccount. Both halves must be right: a Role nobody is bound to grants nothing, and a binding to a name that does not exist grants nothing either.

This exercise runs on the cluster, and k8slings asks the API server what the ServiceAccount `reader` may really do (`kubectl auth can-i`). `reader` must be able to `get` and `list` Pods in its namespace, and must **not** be able to `delete` them.
