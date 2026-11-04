# pods5: more than one container

A Pod can run several containers that share the same network and can share volumes. A common pattern is a **sidecar**: a helper container that runs next to the main one, for logging, proxying or syncing files.

Container names must be **unique within a Pod**, because that is how `kubectl logs -c <name>` and `kubectl exec -c <name>` find them. The schema does not check this, so you have to.
