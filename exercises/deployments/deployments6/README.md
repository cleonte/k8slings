# deployments6: a Deployment that cannot go back

A Deployment keeps its old **ReplicaSets** so that `kubectl rollout undo` has something to go back to. **`revisionHistoryLimit`** says how many it keeps (10 by default); with `0` the old ones are deleted at once, and a rollback has nothing to roll back to. **`minReadySeconds`** is how long a new Pod must stay Ready before it counts as available, so that a Pod that crashes after a moment does not pass for healthy.

The team needs to be able to go back to either of the **last two** versions, and a new Pod has to stay Ready for **ten seconds** before it counts.
