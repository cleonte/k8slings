# rbac5: subresources are resources of their own

Reading a Pod and reading its logs are different permissions: logs are the **subresource** `pods/log`, and a rule for `pods` does not include it (nor `pods/exec`, nor `pods/portforward`). A person who may `get` Pods but gets "forbidden" from `kubectl logs` has a Role without it.

This Role is for the `reader` ServiceAccount, which should read Pods and their logs, and do nothing else.
