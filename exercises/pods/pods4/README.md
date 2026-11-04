# pods4: restart policy

When a container exits, the kubelet decides what to do from the Pod's `spec.restartPolicy`:

- `Always` (the default): restart it, whatever the exit code.
- `OnFailure`: restart it only if it exited with an error.
- `Never`: leave it stopped.

This file is valid YAML and a valid Pod, so nothing rejects it: the problem is in what it does. The container is a one-off task. Make sure Kubernetes does not keep restarting it after it succeeds.
