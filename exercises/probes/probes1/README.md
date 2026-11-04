# probes1: a readiness probe that can pass

A **readiness probe** tells Kubernetes when a container is ready for traffic. Until it succeeds the Pod is not Ready and Services will not send it requests. A probe that checks a path or port the application does not serve fails forever, and the Pod never becomes Ready, even though the container is running.

`nginx` serves its default page on port 80, at `/`. Keep the readiness probe, but make it able to pass.
