# troubleshooting4: the Pod will not be placed

> **Ticket.** The `web` Deployment in the namespace `k8slings-troubleshooting4` has one replica and that Pod has not started since the Deployment was created. Make it run.

This is a **fix** exercise: there is no file to edit. k8slings has broken a Deployment in the cluster on purpose, and you repair it with `kubectl`. `k8slings watch` turns green by itself when it works; `k8slings run troubleshooting4` checks once.

Rules:

- **Repair the Deployment that is there.** Deleting it and creating a new one does not count.
- **Keep one replica and the image.** A Pod that is not wanted is not a repair.

```sh
k8slings kubectl -n k8slings-troubleshooting4 get pods
k8slings kubectl -n k8slings-troubleshooting4 describe pod <the pod>
k8slings kubectl get nodes
```

The Pod is not crashing: it never started. Which part of the cluster decides where a Pod runs, and what does the event say it is missing? `kubectl set resources` and `kubectl edit` change a live Deployment.

`k8slings reset troubleshooting4` puts the cluster back as it was.
