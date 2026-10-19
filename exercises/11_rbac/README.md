# 11 RBAC

RBAC answers "who can do what". A **Role** lists allowed verbs on resources; a **RoleBinding** attaches it to a subject.

## Goal
ServiceAccount `reader` can `get` and `list` pods in its namespace, and must NOT be able to `delete` them.

## Task
Fix the Role and the RoleBinding.
