# daemonsets4: a node agent that uses the node itself

A log collector or a network agent often has to look at the **node**, not at its own container. `hostNetwork: true` puts the Pod in the node's network namespace; a `hostPath` volume mounts a directory of the node's file system. Both are powerful, and each has a rule that goes with it:

- A Pod with `hostNetwork: true` uses the node's DNS settings, so it cannot resolve cluster names such as `web.default.svc` unless it sets `dnsPolicy: ClusterFirstWithHostNet`.
- A `hostPath` that the agent only reads has to be mounted `readOnly: true`, or a bug (or an attacker) in the agent can write to the node's own files.

This DaemonSet collects the node's logs from `/var/log` and sends them to a Service in the cluster.
