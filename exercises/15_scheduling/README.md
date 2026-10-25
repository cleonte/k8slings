# 15 Scheduling constraints

The scheduler only places a Pod on a node that satisfies its `nodeSelector` and its required `nodeAffinity`. If no node does, the Pod stays `Pending` with a `FailedScheduling` event.

## Goal
Both replicas of Deployment `batch` are Ready, and the Deployment keeps both a `nodeSelector` and a required `nodeAffinity`.

## Task
The pods never schedule. Read the scheduler's reason (`kubectl describe pod`) and look at the real node labels (`kubectl get nodes --show-labels`). Fix the constraints so they match this cluster. Do not just delete them.
