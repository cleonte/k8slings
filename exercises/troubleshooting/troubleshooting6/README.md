# troubleshooting6: forbidden

> **Ticket.** The ServiceAccount `reader` in the namespace `k8slings-troubleshooting6` is meant to read Pods, and a Role and a RoleBinding were created for it. It still gets `Forbidden`. Fix it, without giving it more than reading.

This is a **fix** exercise: there is no file to edit. k8slings has set up the namespace on purpose, and you repair it with `kubectl`. `k8slings watch` turns green by itself when it works; `k8slings run troubleshooting6` checks once.

Rules:

- **Repair the RoleBinding that is there.** Deleting it and creating a new one does not count.
- **Least privilege.** `reader` may get and list Pods and nothing else: widening the Role is not a repair.

You can ask the API server what an identity may do:

```sh
k8slings kubectl -n k8slings-troubleshooting6 auth can-i get pods --as system:serviceaccount:k8slings-troubleshooting6:reader
k8slings kubectl -n k8slings-troubleshooting6 describe rolebinding
k8slings kubectl -n k8slings-troubleshooting6 get serviceaccounts
```

Read the RoleBinding slowly: who is it for, and what does it grant?

`k8slings reset troubleshooting6` puts the cluster back as it was.
