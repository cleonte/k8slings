# statefulsets5: the governing Service

A StatefulSet gives each Pod a stable DNS name (`db-0.db`, `db-1.db`) through the Service named in `serviceName`. That only works if the Service is **headless** (`clusterIP: None`): a normal Service has one virtual IP and no per-Pod records.

This StatefulSet runs two database Pods that find each other by name.
