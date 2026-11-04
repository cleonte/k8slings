# namespaces1: putting an object in a namespace

A **namespace** groups objects and gives their names a scope: two Pods can both be called `web` if they live in different namespaces. An object says where it lives with `metadata.namespace`; if it does not, it goes into the namespace you are working in (usually `default`).

A namespace must exist before objects can be created in it, and the name has to match exactly. (In the cluster exercises later on, k8slings picks the namespace for you, so those manifests leave `metadata.namespace` out. Here, the namespace is the lesson.)
