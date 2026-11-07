# troubleshooting15: a delete that never finishes

> **Ticket.** In the namespace `k8slings-troubleshooting15`, remove the old ConfigMap `legacy`. `kubectl delete` does not return an error, and the object is still there.

This is a **fix** exercise: there is no file to edit. k8slings has broken the namespace on purpose, and you repair it with `kubectl`. `k8slings watch` turns green by itself when it works; `k8slings run troubleshooting15` checks once.

Rules:

- **Remove `legacy` and only `legacy`.** The ConfigMap `current` has to stay, as it is.
- Look at what holds the object before you change anything.

```sh
k8slings kubectl -n k8slings-troubleshooting15 delete configmap legacy --wait=false
k8slings kubectl -n k8slings-troubleshooting15 get configmap legacy -o yaml
```

What does `metadata.deletionTimestamp` say, and what is in `metadata.finalizers`? Who is supposed to remove it?

`k8slings reset troubleshooting15` puts the cluster back as it was.
