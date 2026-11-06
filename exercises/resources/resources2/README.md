# resources2: a namespace quota

A **ResourceQuota** caps the total a namespace may use. A **LimitRange** gives containers that set no resources of their own *default* requests and limits. Together they hide a trap: a container with no requests silently gets the LimitRange default, and the quota is checked against that.

When a quota would be exceeded, the *ReplicaSet* is refused when it tries to create the Pod. The Deployment just sits at fewer replicas: there is no failing Pod to describe. The reason is in the namespace's events.

This exercise runs on the cluster. The Deployment `web` should have 3 ready replicas, and you must not change the quota. Work out the arithmetic: the quota allows 500m of CPU requests in total.

```sh
k8slings kubectl -n k8slings-resources2 describe resourcequota team-quota
k8slings kubectl -n k8slings-resources2 get events
```
