# disruption3: a budget that protects nothing

A PodDisruptionBudget finds its Pods with a label selector, exactly like a Service does. If the selector matches no Pod, the budget is valid, Kubernetes accepts it, and it protects nothing: a drain evicts the Pods without ever asking it.

This budget is meant to keep two of the three `web` Pods up during a drain.
