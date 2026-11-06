# security1: not as root

By default a container runs as whatever user its image says, which is often **root**. A `securityContext` lets you say otherwise. `runAsNonRoot: true` makes the kubelet refuse to start a container that would run as root, and `runAsUser` picks the numeric user.

The two settings must agree: a Pod that demands a non-root user *and* asks for user `0` (root) can never start. Kubernetes accepts the manifest; the container just never runs ("container has runAsNonRoot and image will run as root").

Make this Pod run as a non-root user, and keep the guarantee that it never runs as root.
