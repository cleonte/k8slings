# troubleshooting18: a Job that gave up

> **Ticket.** In the namespace `k8slings-troubleshooting18`, the Job `migrate` is shown as Failed, and the migration has not been done.

This is a **fix** exercise: there is no file to edit. k8slings has broken the namespace on purpose, and you repair it with `kubectl`. `k8slings watch` turns green by itself when it works; `k8slings run troubleshooting18` checks once.

Rules:

- **Run the migration script of the ConfigMap `migrate-script`.** Replacing the work with something else, so that the Job passes, is not a repair; neither is changing the ConfigMap.
- A Job's Pod template cannot be changed once it has been created. What does that leave you?

```sh
k8slings kubectl -n k8slings-troubleshooting18 get job,pods
k8slings kubectl -n k8slings-troubleshooting18 logs job/migrate
k8slings kubectl -n k8slings-troubleshooting18 get job migrate -o yaml
```

Read the log of the failed Pod: it says what went wrong. Then ask whether raising `backoffLimit` can help.

`k8slings reset troubleshooting18` puts the cluster back as it was.
