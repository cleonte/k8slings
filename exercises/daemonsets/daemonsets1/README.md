# daemonsets1: one Pod on every node

A **DaemonSet** runs one Pod on every node (or on every node that matches a selector): log collectors, monitoring agents, network plugins. New nodes get the Pod automatically.

"Every node" is subject to **taints**. A taint on a node repels Pods unless they carry a matching **toleration**. Control-plane nodes are tainted so that ordinary workloads do not land on them, which means a DaemonSet with no tolerations runs on the workers only.

This exercise needs the multi-node cluster (the default of `k8slings cluster up`). The `agent` DaemonSet is meant to run on **every** node, the control plane too, and it does not. Look at the nodes, and at what stops it:

```sh
k8slings kubectl get nodes
k8slings kubectl describe node <the control-plane node> | grep -i taint
k8slings kubectl -n k8slings-daemonsets1 get daemonset agent
```
