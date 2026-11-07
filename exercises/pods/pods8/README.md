# pods8: a Pod that learns about itself

The **downward API** gives a container information about its own Pod without asking the API server: the Pod's name, namespace and node, its labels, and its own resource limits. In an environment variable it is `valueFrom` with a `fieldRef` (a field of the Pod, such as `metadata.name`) or a `resourceFieldRef` (a request or limit of a container). Writing the values by hand goes wrong the first time the Pod is renamed or moved.

The container needs four variables: `POD_NAME`, `POD_NAMESPACE` and `NODE_NAME` with what the Pod really is, and `MEM_LIMIT` with the memory limit of the container. Keep the memory limit.
