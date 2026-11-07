# volumes3: three volumes for one directory

Only one volume can be mounted on a path, and the API server rejects a Pod that mounts two on the same one. When an application wants a ConfigMap, a Secret and a few facts about the Pod **in the same directory**, a **projected volume** merges several sources (`configMap`, `secret`, `downwardAPI`, `serviceAccountToken`) into one.

The container reads everything from `/etc/app`. Make the Pod valid with a single volume, keeping all three sources.
