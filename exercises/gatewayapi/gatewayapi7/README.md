# gatewayapi7: a canary that gets nine requests in ten

When a rule has several `backendRefs`, each gets a share of the requests proportional to its `weight` (default 1). Weights are relative, not percentages: 90 and 10 send nine requests in ten to the first. A canary release sends a **small** share to the new version, and `weight: 0` sends it none at all.

This needs the gateway add-on: `k8slings cluster addon install gateway`.

`v1` is the stable version and `v2` the canary. The route below sends the canary nine requests in ten. Make the canary receive about **one request in ten**, and keep both versions in the route.
