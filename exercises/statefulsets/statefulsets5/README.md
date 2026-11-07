# statefulsets5: what happens to the volumes

The PersistentVolumeClaims that a StatefulSet creates from its `volumeClaimTemplates` are **kept** when it is scaled down or deleted: data is precious. `persistentVolumeClaimRetentionPolicy` changes that, separately for the two cases:

- `whenScaled`: what happens to the claims of the replicas that a scale-down removes.
- `whenDeleted`: what happens to all the claims when the StatefulSet itself is deleted.

Each is `Retain` (the default) or `Delete`.

This StatefulSet is a cache that is rebuilt from the database. Its volumes are worth nothing once the StatefulSet is deleted, but while it exists a scale-down must **not** throw away the volumes: they are reused when it scales up again, and they hold the warm data.
