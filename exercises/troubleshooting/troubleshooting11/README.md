# troubleshooting11: nowhere to go

> **Ticket.** The `web` Deployment in the namespace `k8slings-troubleshooting11` has to run on one specific worker node, and its Pod has never been placed. Make it run.

This is a **fix** exercise: there is no file to edit. k8slings has broken the namespace on purpose, and you repair it with `kubectl`. `k8slings watch` turns green by itself when it works; `k8slings run troubleshooting11` checks once.

Rules:

- **Repair the Deployment that is there.** Deleting it and creating a new one does not count.
- **Keep the placement rule.** The Pod stays tied to a node by its affinity: removing the rule, or bypassing the scheduler with `nodeName`, is not a repair.
- **Leave the nodes alone.** The cluster is shared with the other exercises. If you changed a node's hostname label, a reset will not undo it: `k8slings kubectl label node <name> kubernetes.io/hostname=<name> --overwrite`.

```sh
k8slings kubectl -n k8slings-troubleshooting11 get pods
k8slings kubectl -n k8slings-troubleshooting11 describe pod <the pod>
k8slings kubectl get nodes --show-labels
```

The event says how many nodes were looked at and why each was rejected. Compare what the Pod asks for with what the nodes really are.

`k8slings reset troubleshooting11` puts the cluster back as it was.
