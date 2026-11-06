# troubleshooting2: the new release never starts

> **Ticket.** The `web` Deployment in the namespace `k8slings-troubleshooting2` was updated this morning and none of its Pods has started since. The image is meant to be the official nginx, version 1.27. Make it run again.

This is a **fix** exercise: there is no file to edit. k8slings has broken a Deployment in the cluster on purpose, and you repair it with `kubectl`. `k8slings watch` looks at the cluster every couple of seconds and turns green when it is repaired; `k8slings run troubleshooting2` checks once.

Rules:

- **Repair the Deployment that is there.** Deleting it and creating a new one does not count.
- **Keep two replicas.**

Start with the Pods:

```sh
k8slings kubectl -n k8slings-troubleshooting2 get pods
k8slings kubectl -n k8slings-troubleshooting2 describe pod <a pod>
```

The Pods never get to run a container. What state are they in, and what do the events at the bottom of `describe` say? To change the image of a live Deployment: `kubectl set image` or `kubectl edit`.

`k8slings reset troubleshooting2` puts the cluster back as it was.
