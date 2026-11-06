# security4: the "restricted" Pod Security Standard

**Pod Security Admission** is built into Kubernetes. A namespace can be labelled to *enforce* one of three Pod Security Standards, and the API server then rejects any Pod that does not meet it:

- `privileged`: no restrictions.
- `baseline`: blocks the known dangerous settings (privileged containers, host namespaces, ...).
- `restricted`: current hardening best practice. On top of baseline, a Pod must:
  - run as a non-root user (`runAsNonRoot: true`),
  - not allow privilege escalation (`allowPrivilegeEscalation: false`),
  - drop **all** capabilities (and add back at most `NET_BIND_SERVICE`),
  - use a seccomp profile (`RuntimeDefault` is the usual one).

A namespace enforcing `restricted` is labelled `pod-security.kubernetes.io/enforce: restricted`. This Pod would be rejected by it. Make it satisfy the restricted standard. k8slings checks each control from the list above.
