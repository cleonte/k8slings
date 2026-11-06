# troubleshooting3: the Service has nobody behind it

> **Ticket.** In the namespace `k8slings-troubleshooting3`, the Pods of `web` are Running and Ready, but nothing answers on the Service `web`. Make the Service reach them.

This is a **fix** exercise: there is no file to edit. k8slings has broken the namespace on purpose, and you repair it with `kubectl`. `k8slings watch` turns green by itself when it works; `k8slings run troubleshooting3` checks once.

Rules:

- **Repair the Service that is there.** Deleting it and creating a new one does not count.
- **Do not touch the Deployment.** Its Pods are fine.

A Service sends traffic to the Pods that are listed as its endpoints. Where do you see them?

```sh
k8slings kubectl -n k8slings-troubleshooting3 get pods --show-labels
k8slings kubectl -n k8slings-troubleshooting3 get service web -o wide
k8slings kubectl -n k8slings-troubleshooting3 get endpointslices
```

`k8slings reset troubleshooting3` puts the cluster back as it was.
