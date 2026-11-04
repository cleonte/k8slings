# 13 Persistent storage

A **PersistentVolumeClaim** asks for storage. A **StorageClass** says how it is provisioned. The claim must name a class that exists, and an access mode that class's provisioner supports.

## Goal
PVC `notes` is `Bound`, and pod `writer` is Running with the volume mounted at `/data` (it logs `saved`).

## Task
The claim never binds. Look at the available classes (`kubectl get storageclass`) and at the claim's events, then fix the PVC. There may be more than one problem.
