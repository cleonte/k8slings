# deployments5: a rollout that waits for itself

A rolling update starts the new Pod **before** it stops the old one. With a volume that may be used by **one Pod only** (`ReadWriteOncePod`), the new Pod cannot start until the old one has let go of the claim, and the old one is only stopped when the new one is Ready. The rollout waits for ever.

The single replica of this Deployment must never run two Pods at once. Fix the strategy; do not touch the claim.
