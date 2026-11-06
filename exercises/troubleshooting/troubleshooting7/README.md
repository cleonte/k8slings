# troubleshooting7: nothing gets created

> **Ticket.** The `web` Deployment in the namespace `k8slings-troubleshooting7` should run two replicas, but no Pod ever appears. Make it work.

This is a **fix** exercise: there is no file to edit. k8slings has broken the namespace on purpose, and you repair it with `kubectl`. `k8slings watch` turns green by itself when it works; `k8slings run troubleshooting7` checks once.

Rules:

- **Repair the Deployment that is there.** Deleting it and creating a new one does not count.
- **Leave the ResourceQuota alone.** It is the team's budget for this namespace: changing or deleting it is not a repair.
- **Keep two replicas.**

Two repairs are accepted: requests on the Deployment, or a LimitRange in the namespace that gives every container default requests.

```sh
k8slings kubectl -n k8slings-troubleshooting7 get deployment web
k8slings kubectl -n k8slings-troubleshooting7 describe replicaset
k8slings kubectl -n k8slings-troubleshooting7 describe resourcequota
```

There are no Pods to describe, so look one level up: who creates the Pods, and what does it say when it cannot? Then read what the quota asks of every Pod.

`k8slings reset troubleshooting7` puts the cluster back as it was.
