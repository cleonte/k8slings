# troubleshooting10: waiting for a disk

> **Ticket.** The `web` Deployment in the namespace `k8slings-troubleshooting10` keeps its data on a PersistentVolumeClaim, and its Pod has never started. Make it run, with its data on the volume.

This is a **fix** exercise: there is no file to edit. k8slings has broken the namespace on purpose, and you repair it with `kubectl`. `k8slings watch` turns green by itself when it works; `k8slings run troubleshooting10` checks once.

Rules:

- **Keep the Deployment as it is.** It is configured the way the team wants: the volume stays a PersistentVolumeClaim called `data`.
- **Keep a claim called `data`.** It has to end up Bound.
- **Fix the claim, not the cluster.** Do not create the StorageClass it asks for. (If you did, `k8slings kubectl delete storageclass fast-ssd` puts the cluster back: a reset does not touch cluster-wide objects.)

```sh
k8slings kubectl -n k8slings-troubleshooting10 get pods,pvc
k8slings kubectl -n k8slings-troubleshooting10 describe pvc data
k8slings kubectl get storageclass
```

Some fields of an object cannot be changed once it exists (`kubectl patch` will tell you). When that is the one at fault, the object has to be replaced, and that is fine for a claim that nothing has written to.

`k8slings reset troubleshooting10` puts the cluster back as it was.
