# troubleshooting5: Running, but not Ready

> **Ticket.** The Pods of `web` in the namespace `k8slings-troubleshooting5` are Running and nothing restarts, yet the Deployment never gets its two replicas available and the Service sends them no traffic. Make them Ready.

This is a **fix** exercise: there is no file to edit. k8slings has broken a Deployment in the cluster on purpose, and you repair it with `kubectl`. `k8slings watch` turns green by itself when it works; `k8slings run troubleshooting5` checks once.

Rules:

- **Repair the Deployment that is there.** Deleting it and creating a new one does not count.
- **Keep the readiness probe.** A Pod that reports Ready without being asked is not a repair, and it is how bad releases reach users.
- **Keep two replicas and the image.**

```sh
k8slings kubectl -n k8slings-troubleshooting5 get pods
k8slings kubectl -n k8slings-troubleshooting5 describe pod <a pod>
k8slings kubectl -n k8slings-troubleshooting5 logs <a pod>
```

What do the events say about the probe, and what does the application's own log say about the request it got?

`k8slings reset troubleshooting5` puts the cluster back as it was.
