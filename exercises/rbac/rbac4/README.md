# rbac4: a binding that cannot find its ServiceAccount

ServiceAccounts live in a namespace, and a RoleBinding can be in another one. So a subject that is a ServiceAccount has to say **which namespace** the account is in; the name alone is not enough, and the API server rejects the binding.

The ServiceAccount `deployer` lives in `ci`; the Role is in `apps`. Make the binding valid, and point it at the account that exists.
