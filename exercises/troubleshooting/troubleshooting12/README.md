# troubleshooting12: back to yesterday

> **Ticket.** This morning's release broke `web` in the namespace `k8slings-troubleshooting12`. Get it back to what was running yesterday.

This is a **fix** exercise: there is no file to edit. k8slings has deployed two releases of a Deployment, the second one broken, and you repair it with `kubectl`. `k8slings watch` turns green by itself when it works; `k8slings run troubleshooting12` checks once.

Rules:

- **Repair the Deployment that is there.** Deleting it and creating a new one does not count.
- **Keep two replicas.**
- **Run yesterday's version**, not a different one.

A Deployment remembers its earlier versions, and Kubernetes can go back to one of them:

```sh
k8slings kubectl -n k8slings-troubleshooting12 get pods
k8slings kubectl -n k8slings-troubleshooting12 rollout status deployment/web
k8slings kubectl -n k8slings-troubleshooting12 rollout history deployment/web
```

What is different between the revisions? (`rollout history --revision=1` shows one.) Then look at `kubectl rollout undo`.

Setting the old image back by hand also works: the check cannot tell the two apart. But on a real incident, going back is the first thing to try, and you will have the Deployment's history to thank for it.

`k8slings reset troubleshooting12` deploys both releases again.
