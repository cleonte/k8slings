# troubleshooting16: a slow starter that is killed

> **Ticket.** In the namespace `k8slings-troubleshooting16`, the Pod of the Deployment `web` never stays up: it restarts again and again, and nobody has changed the application.

This is a **fix** exercise: there is no file to edit. k8slings has broken the namespace on purpose, and you repair it with `kubectl`. `k8slings watch` turns green by itself when it works; `k8slings run troubleshooting16` checks once.

Rules:

- **Keep the liveness probe.** A container that hangs must still be restarted.
- **The application is not what has to change.** Do not touch its command.
- **Repair the Deployment that is there.** Deleting it and creating a new one does not count.

```sh
k8slings kubectl -n k8slings-troubleshooting16 get pods
k8slings kubectl -n k8slings-troubleshooting16 describe pod -l app=web
k8slings kubectl -n k8slings-troubleshooting16 logs -l app=web --previous
```

Look at the restart count, and at the events at the bottom of `describe`. The same pattern is the lesson of the offline exercise `probes2`; here you meet it as it looks live.

`k8slings reset troubleshooting16` puts the cluster back as it was.
