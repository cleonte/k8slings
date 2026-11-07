# troubleshooting13: the client cannot find anything

> **Ticket.** In the namespace `k8slings-troubleshooting13`, the Deployment `client` runs, but from inside its Pod no name resolves: `wget http://web` fails with "bad address". The Service `web` is fine and has an endpoint.

This is a **fix** exercise: there is no file to edit. k8slings has broken the namespace on purpose, and you repair it with `kubectl`. `k8slings watch` turns green by itself when it works; `k8slings run troubleshooting13` checks once.

Rules:

- **Repair the Deployment that is there.** Deleting it and creating a new one does not count.
- **Let DNS resolve the name.** A hand-written hosts entry for the Service does not count.

```sh
k8slings kubectl -n k8slings-troubleshooting13 get pods,services
k8slings kubectl -n k8slings-troubleshooting13 exec deploy/client -- cat /etc/resolv.conf
k8slings kubectl -n k8slings-troubleshooting13 exec deploy/client -- nslookup web
```

The Service is not the suspect. Where does a Pod learn which DNS server to ask, and who decides that?

`k8slings reset troubleshooting13` puts the cluster back as it was.
