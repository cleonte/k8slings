# autoscaling5: a percentage of nothing

A `Utilization` target is a percentage **of the Pod's request**: 70% of 100m of CPU is 70m. A container that requests nothing has no 100%, so the autoscaler has nothing to compute, shows `<unknown>` as its current value, reports `ScalingActive: False` and never scales, however busy the Pods are. The metrics (from the metrics-server add-on) are there; the percentage is not.

This needs the metrics-server add-on: `k8slings cluster addon install metrics-server`.

The autoscaler below should be able to read the CPU use of the Pods of `web` as a percentage, and so be **active**. Keep the Utilization target.
