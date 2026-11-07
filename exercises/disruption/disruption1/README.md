# disruption1: a budget that forbids every disruption

A **PodDisruptionBudget** limits how many Pods of an application may be taken down **voluntarily** (a node drain, a cluster upgrade) at the same time. `minAvailable` is how many must stay up; `maxUnavailable` is how many may be down. A budget that demands that *all* the replicas stay up allows no disruption at all: every drain of a node with one of these Pods waits for ever, and the upgrade of the cluster gets stuck.

The Deployment runs three replicas. A drain has to be able to take one of them down, but never two. Make the budget say that.
