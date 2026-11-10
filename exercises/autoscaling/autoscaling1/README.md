# autoscaling1: an autoscaler that points at nothing

A HorizontalPodAutoscaler changes the replica count of another object, and finds it with `scaleTargetRef`: the **full address** of the object, `apiVersion`, `kind` and `name`, the same three fields as in a manifest. A Deployment is `apps/v1`, not `v1`. The API server does not check that the target exists when it accepts the autoscaler, so a wrong address is not an error: the HPA is created and then never does anything (its condition `AbleToScale` says why).

This autoscaler is meant to scale the Deployment `web`.
