# disruption4: minAvailable and maxUnavailable together

A PodDisruptionBudget says how much of the group may go with **one** of two fields: `minAvailable` (how many must stay) or `maxUnavailable` (how many may leave). Setting both is rejected by the API server, because the two can contradict each other.

This budget is meant to let a drain take one of the three Pods at a time.
