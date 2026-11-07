# troubleshooting14: a change that nobody sees

> **Ticket.** In the namespace `k8slings-troubleshooting14`, the feature flag `FEATURE_FLAG` was switched in the ConfigMap `app-config` an hour ago, and the application `web` still behaves as if it were off.

This is a **fix** exercise: there is no file to edit. k8slings has broken the namespace on purpose, and you repair it with `kubectl`. `k8slings watch` turns green by itself when it works; `k8slings run troubleshooting14` checks once.

Rules:

- **Keep the new value of the flag.** Setting it back is not a repair.
- **The application keeps reading the flag from the ConfigMap.** Writing the value into the Deployment does not count, and neither does deleting the Deployment and creating a new one.

```sh
k8slings kubectl -n k8slings-troubleshooting14 get configmap app-config -o yaml
k8slings kubectl -n k8slings-troubleshooting14 exec deploy/web -- printenv FEATURE_FLAG
```

The ConfigMap says one thing and the Pod says another. When does a container read an environment variable?

`k8slings reset troubleshooting14` puts the cluster back as it was.
