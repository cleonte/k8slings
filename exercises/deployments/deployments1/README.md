# deployments1: scale and change the image

A **Deployment** keeps a number of identical Pods running (`spec.replicas`) and replaces them gradually when the Pod template changes, for example a new image. You change both by editing the manifest: Kubernetes works out the difference.

Run 3 replicas of the `api` Deployment on `nginx:1.27`.
