# troubleshooting20: a Pod that waits for a node it may not use

> **Ticket.** In the namespace `k8slings-troubleshooting20`, the Deployment `ctl` has to run on the **control-plane** node (it is a controller for the cluster itself, and its `nodeSelector` says so). Its Pod is Pending and nothing is wrong with the image.

This is a **fix** exercise: there is no file to edit. k8slings has broken the namespace on purpose, and you repair it with `kubectl`. `k8slings watch` turns green by itself when it works; `k8slings run troubleshooting20` checks once.

Rules:

- **The Pod runs on the control-plane node.** Keep its `nodeSelector`, and do not pin it with `nodeName`.
- **The node keeps its taint.** The Pod has to be allowed in, by a toleration that names this taint; a toleration that matches every taint does not count, and the node must not be opened to every Pod. (Needs the cluster with its workers: the control-plane node is only tainted there.)
- **Repair the Deployment that is there.** Deleting it and creating a new one does not count.

```sh
k8slings kubectl -n k8slings-troubleshooting20 get pods
k8slings kubectl -n k8slings-troubleshooting20 describe pod -l app=ctl
k8slings kubectl describe nodes -l node-role.kubernetes.io/control-plane
```

If you did remove the taint from the node, put it back with
`k8slings kubectl taint nodes -l node-role.kubernetes.io/control-plane node-role.kubernetes.io/control-plane:NoSchedule --overwrite`
(`reset` re-creates the Deployment, but does not touch nodes).

`k8slings reset troubleshooting20` puts the cluster back as it was.
