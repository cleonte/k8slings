# serviceaccounts3: no token mounted

A Pod talks to the API server with the token of its ServiceAccount, which Kubernetes mounts into it. A ServiceAccount can switch that off with `automountServiceAccountToken: false` (a good default for Pods that never call the API). A Pod that **does** need the API then has to ask for the token itself, or the ServiceAccount must stop refusing it.

This controller lists Pods through the API, with the ServiceAccount `lister`.
