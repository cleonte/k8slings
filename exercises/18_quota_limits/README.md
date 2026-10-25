# 18 Quotas and default limits

A **ResourceQuota** caps the total a namespace can use. A **LimitRange** fills in defaults for containers that do not set their own. When a quota would be exceeded, the *ReplicaSet* is refused creating the pod: the Deployment just sits at fewer replicas, and there is no failing Pod to describe.

## Goal
Deployment `web` has 3 Ready replicas, with the quota left as it is.

## Task
Only some replicas come up. Find where the error is reported (`kubectl describe` the ReplicaSet, or the namespace events), work out the arithmetic against the quota, and fix the Deployment. Do not change the quota.
