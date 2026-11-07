# volumes4: a claim that nothing can provide

A PersistentVolumeClaim asks for an **access mode**: `ReadWriteOnce` (one node at a time), `ReadWriteOncePod` (one Pod), `ReadOnlyMany` or `ReadWriteMany` (many nodes at once). The StorageClass has to be able to provide it. The default class of this cluster keeps each volume on the disk of one node, so it cannot offer every mode; the claim then waits and never binds, and the Pod that wants it never starts.

The Pod `app` stays Pending. Read what the cluster says about the claim, and make the Pod start while its data stays on a PersistentVolumeClaim.
