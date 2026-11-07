# probes4: a named port that is not there

A probe can point at a container port by **name** (`port: http`) instead of by number, which keeps the two in step when the number changes. But the name has to exist in the container's `ports`; if it does not, the kubelet cannot find where to probe and the container is reported unhealthy.

This container listens on 8080, and names the port `web`.
