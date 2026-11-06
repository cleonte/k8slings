# troubleshooting1: it keeps restarting

> **Ticket.** The `web` Deployment in the namespace `k8slings-troubleshooting1` should run two replicas, but its Pods keep restarting and never become ready. Nobody changed the image. Make it work.

This is a **fix** exercise: there is no file to edit. k8slings has just broken a Deployment in the cluster on purpose, and you repair it with `kubectl`, the way you would on a real incident. The full-screen view (`k8slings watch`) looks at the cluster every couple of seconds and turns green when it is repaired. Without it, run `k8slings run troubleshooting1` to check.

Two rules, because a real system cannot just be thrown away:

- **Repair the Deployment that is there.** Deleting it and creating a new one is noticed, and does not count.
- **Keep the image and the number of replicas.** The fault is somewhere else.

Work from the outside in:

```sh
k8slings kubectl -n k8slings-troubleshooting1 get pods
k8slings kubectl -n k8slings-troubleshooting1 describe pod <a pod>
k8slings kubectl -n k8slings-troubleshooting1 logs <a pod>
```

What state are the Pods in? What does the container say before it dies? Then change what is wrong: `kubectl edit`, `kubectl set ...` and `kubectl patch` all work on the live object.

If you break it worse, `k8slings reset troubleshooting1` puts the cluster back as it was.
