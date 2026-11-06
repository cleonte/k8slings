# statefulsets1: one volume per replica

A **StatefulSet** gives each replica a stable identity (`db-0`, `db-1`, ...) and, with `volumeClaimTemplates`, **its own volume**: Kubernetes creates one PersistentVolumeClaim per Pod from the template, and the claim stays with that Pod when it is rescheduled or restarted.

Pointing every replica at one shared claim (`persistentVolumeClaim` under the Pod's `volumes`) is the mistake to avoid: with the usual `ReadWriteOnce` access mode, only one node can mount it, and the replicas would write over each other even if they could.

Give each replica of this database its own volume.
