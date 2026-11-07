# troubleshooting19: an application with no shell

> **Ticket.** In the namespace `k8slings-troubleshooting19`, the Service `web` has an endpoint, and every request to it fails. The Deployment looks healthy.

This is a **fix** exercise: there is no file to edit. k8slings has broken the namespace on purpose, and you repair it with `kubectl`. `k8slings watch` turns green by itself when it works; `k8slings run troubleshooting19` checks once.

Rules:

- **Keep the application as it is.** Its image has no shell on purpose; swapping it for an image that has one does not count.
- **Repair the Deployment that is there.** Deleting it and creating a new one does not count.

```sh
k8slings kubectl -n k8slings-troubleshooting19 get pods,service,endpoints
k8slings kubectl -n k8slings-troubleshooting19 exec deploy/web -- sh
```

That last command fails: there is nothing in the image to run. A **debug container** next to it shares the Pod's network and, with `--target`, its processes:

```sh
k8slings kubectl -n k8slings-troubleshooting19 debug -it POD --image=busybox:1.37 --target=web -- sh
```

In a terminal that opens a shell. Without one, start a container that waits (`... debug POD --image=busybox:1.37 --target=web --container=dbg -q -- sleep 600`) and run `kubectl exec POD -c dbg -- netstat -ln`.

What is the application really listening on?

`k8slings reset troubleshooting19` puts the cluster back as it was.
