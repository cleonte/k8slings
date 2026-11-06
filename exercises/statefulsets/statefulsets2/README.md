# statefulsets2: a stable name for each Pod

The Pods of a StatefulSet have stable names, and with a **headless Service** (`clusterIP: None`) each Pod also gets a stable DNS name of its own: `web-0.web`, `web-1.web`, ... A normal Service gives you one virtual IP that load-balances over the Pods; a headless one publishes a DNS record **per Pod**, so clients can address a particular replica (the primary of a database, say). The StatefulSet's `serviceName` must name that Service.

This exercise runs on the cluster. A client Pod tries to reach `web-0` by its own name, and fails. Find out why with `k8slings kubectl -n k8slings-statefulsets2 ...` (for example `exec client -- nslookup web-0.web`), and fix the Service.

A note on timing: DNS remembers an answer for a while, including "there is no such name". After you fix the Service it can take up to half a minute until `web-0.web` resolves, so give it a moment.
