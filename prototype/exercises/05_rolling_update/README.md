# 05 Rolling update strategy

`Recreate` kills every old pod before starting new ones: downtime. `RollingUpdate` replaces pods gradually.

## Goal
Deployment `shop`: 3 replicas, strategy `RollingUpdate` with `maxUnavailable: 0` and `maxSurge: 1`
(never drop below 3 serving pods, add at most 1 extra during a rollout).

## Task
Fix the `strategy` block.
