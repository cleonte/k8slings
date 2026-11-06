# scheduling1: constraints that match real nodes

The scheduler only puts a Pod on a node that satisfies its `nodeSelector` and its **required** `nodeAffinity`. If no node does, the Pod stays `Pending`, and the reason is in its events (`FailedScheduling`).

Both constraints refer to **node labels**, so they have to be labels that the nodes really have. This exercise needs a cluster with several nodes (the default of `k8slings cluster up`).

The Deployment `batch` never gets its Pods scheduled. Find out why, and fix the constraints so that they match this cluster. Keep both the `nodeSelector` and the required `nodeAffinity`: fix them, do not delete them. Useful commands:

```sh
k8slings kubectl get nodes --show-labels
k8slings kubectl -n k8slings-scheduling1 describe pods
```

The Pods should run on the worker nodes. Control-plane nodes carry the label `node-role.kubernetes.io/control-plane`; worker nodes have no role label at all.
