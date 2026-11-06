# volumes1: a claim that binds

A **PersistentVolumeClaim** (PVC) asks for storage. A **StorageClass** says how that storage is provisioned. For the claim to bind, it must name a class that exists, and an access mode that the class's provisioner supports.

This exercise runs on the cluster. The claim `notes` never binds, so the Pod `writer` that mounts it stays Pending. Find out why with the cluster's own tools, for example:

```sh
k8slings kubectl get storageclass
k8slings kubectl -n k8slings-volumes1 describe pvc notes
```

There is more than one problem. Fix the claim, not the Pod.
