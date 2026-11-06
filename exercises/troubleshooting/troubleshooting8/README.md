# troubleshooting8: the client cannot find the server

> **Ticket.** In the namespace `k8slings-troubleshooting8`, the Pod `client` must reach the Service `web` by name, and only that. Since a NetworkPolicy was added, `wget http://web` fails.

This is a **fix** exercise: there is no file to edit. k8slings has broken the namespace on purpose, and you repair it with `kubectl`. `k8slings watch` turns green by itself when it works; `k8slings run troubleshooting8` checks once.

Rules:

- **Repair the NetworkPolicy that is there.** Deleting it and creating a new one does not count.
- **Least privilege.** `client` may talk to `web` and nothing else, except what it needs to find it. The Service `other` must stay out of reach.

```sh
k8slings kubectl -n k8slings-troubleshooting8 get pods,services
k8slings kubectl -n k8slings-troubleshooting8 describe networkpolicy
k8slings kubectl -n k8slings-troubleshooting8 exec client -- wget -qO- -T 3 http://web.k8slings-troubleshooting8.svc.cluster.local
```

Read the error carefully: is it the connection that fails, or something that happens before it? What does a Pod need to do before it can connect to a name, and does the policy allow that?

`k8slings reset troubleshooting8` puts the cluster back as it was.
