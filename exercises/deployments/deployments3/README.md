# deployments3: a rollout that can never move

A rolling update replaces Pods in steps. `maxSurge` is how many extra Pods may be started above the replica count; `maxUnavailable` is how many may be missing. If both are zero the rollout could neither start a new Pod nor remove an old one, and the API server rejects the Deployment.

This service must never lose capacity during a rollout: no Pod may be removed before its replacement is Ready. Make the Deployment valid and keep that promise.
