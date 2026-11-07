# capstone2: the site is down

> **Ticket.** In the namespace `k8slings-capstone2`, the site returns errors and the worker never starts. Get both working.

This is a **fix** exercise: there is no file to edit. k8slings has broken the namespace on purpose, and you repair it with `kubectl`. This one is not a single fault: work out what is wrong with each part, one at a time. `k8slings watch` turns green by itself when everything works; `k8slings run capstone2` checks once and tells you about the first thing that is still wrong.

Rules:

- **Repair what is there.** The Deployment `web`, the Service `web`, the ConfigMap `settings` and the Pod `worker` are the originals. Deleting one and creating a new one does not count.
- **Keep two replicas of `web`**, and keep the images the parts run: the faults are somewhere else.

Start from the outside, and take one thing at a time:

```sh
k8slings kubectl -n k8slings-capstone2 get all,configmap
k8slings kubectl -n k8slings-capstone2 describe pod <a pod>
k8slings kubectl -n k8slings-capstone2 get endpointslices
```

`k8slings reset capstone2` puts the cluster back as it was.
