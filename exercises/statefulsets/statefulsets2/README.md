# statefulsets2: the governing Service

The Pods of a StatefulSet have stable names (`web-0`, `web-1`, ...), and each can also have a stable **DNS name** of its own. That works through the StatefulSet's `serviceName`: it must name the **headless Service** (`clusterIP: None`) that governs the set. Kubernetes gives every Pod its name as `hostname` and the `serviceName` as `subdomain`, and the Pod is then reachable as:

```
web-0.web.<namespace>.svc.cluster.local      (pod name . serviceName . namespace . svc . cluster domain)
```

If no Service called like that exists, the name does not resolve, and nothing in the StatefulSet complains.

This exercise runs on the cluster. A client Pod tries to reach `web-0` and `web-1` by their own names, and fails. The Service is fine. Look at what the Pods think their subdomain is, and compare:

```sh
k8slings kubectl -n k8slings-statefulsets2 get pod web-0 -o jsonpath='{.spec.subdomain}'
k8slings kubectl -n k8slings-statefulsets2 get svc
```

The Pods start in parallel (`podManagementPolicy: Parallel`) to keep this exercise quick; by default a StatefulSet starts them one after another.

A note on timing: DNS remembers an answer for a while, including "there is no such name". After you fix the manifest it can take up to half a minute until the name resolves, so give it a moment.
