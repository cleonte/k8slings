# autoscaling4: a scale-down that waits and goes slowly

Load is spiky, and an autoscaler that follows every dip makes the Pods come and go. `spec.behavior` tunes the two directions separately. For `scaleDown`, `stabilizationWindowSeconds` makes the autoscaler look at the **highest** recommendation of the last N seconds instead of the latest one, and `policies` limit the speed: `type: Pods, value: 1, periodSeconds: 60` removes at most one Pod per minute. Scaling up should stay quick, so `scaleUp` keeps its default (no waiting).

This autoscaler should wait **five minutes** before scaling down, and then remove **at most one Pod per minute**.
