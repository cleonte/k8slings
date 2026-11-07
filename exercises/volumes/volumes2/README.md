# volumes2: a mount of a volume that does not exist

A Pod lists its **volumes**, and each container lists where it mounts them (**volumeMounts**). The two are tied only by the **name**. A mount of a name that the Pod does not have is rejected by the API server: the Pod is never created.

The data must stay on the PersistentVolumeClaim (an `emptyDir` is lost with the Pod). Make the Pod valid.
