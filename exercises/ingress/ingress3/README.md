# ingress3: an Ingress that no controller is asked to serve

An Ingress is only a request. A **controller** turns it into routing, and `spec.ingressClassName` says which one: it names an `IngressClass`, and each controller serves the classes that are its own. A cluster can run several controllers at once. An Ingress with no class is served by whichever controller is configured to take them, or by none, and that depends on the cluster, not on the manifest.

This cluster's controller is Traefik, whose IngressClass is called `traefik`. The Ingress below should be served by it, whatever else is installed.
