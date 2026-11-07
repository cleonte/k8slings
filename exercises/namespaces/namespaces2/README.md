# namespaces2: a Pod that cannot see its ConfigMap

Most objects live in a namespace, and a Pod can only use ConfigMaps, Secrets and PersistentVolumeClaims **of its own namespace**: there is no way to name one from another namespace. A Pod in the wrong namespace starts with `CreateContainerConfigError` and the ConfigMap is right there, one namespace over.

The three objects below were meant to be in `team-a`. Fix it.
