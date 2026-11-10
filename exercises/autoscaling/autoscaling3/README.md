# autoscaling3: a percentage written as an amount

A resource metric has a `target` with a `type`, and the type decides which field says the value: `Utilization` is a **percentage of the Pod's request** (`averageUtilization`), `AverageValue` is an **amount** (`averageValue`, such as `200m` of CPU) and `Value` is for the whole object. Writing the field of one type under another is **not** rejected by the API server, which only checks the shape: the autoscaler is created, and then the controller cannot compute anything from a `Utilization` target that has no `averageUtilization`, and reports it in its conditions.

This autoscaler should add Pods when the average CPU use passes **70% of what the Pods request**.
