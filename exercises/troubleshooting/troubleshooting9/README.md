# troubleshooting9: killed without a word

> **Ticket.** The `web` Deployment in the namespace `k8slings-troubleshooting9` runs one Pod that keeps being restarted, and its log shows nothing wrong. Make it stay up.

This is a **fix** exercise: there is no file to edit. k8slings has broken the namespace on purpose, and you repair it with `kubectl`. `k8slings watch` turns green by itself when it works; `k8slings run troubleshooting9` checks once.

Rules:

- **Repair the Deployment that is there.** Deleting it and creating a new one does not count.
- **Keep the program as it is.** It needs the memory it needs.
- **Keep a limit, and keep it sensible.** No limit at all, or one the node cannot honour, is not a repair.

```sh
k8slings kubectl -n k8slings-troubleshooting9 get pods
k8slings kubectl -n k8slings-troubleshooting9 describe pod <the pod>
```

Look at the *last state* of the container in `describe`: who ended it, and with which exit code? 137 is 128 plus a signal. Compare what the container is allowed with what it uses.

`k8slings reset troubleshooting9` puts the cluster back as it was.
