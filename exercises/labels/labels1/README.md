# labels1: a Deployment must select its own Pods

A Deployment finds its Pods with `spec.selector`, and creates them from `spec.template`. The labels in the template must satisfy the selector, otherwise the Deployment would create Pods it can never own. Kubernetes rejects such a Deployment when you apply it.

Make this Deployment valid without changing its selector.
