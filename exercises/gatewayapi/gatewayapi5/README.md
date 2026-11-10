# gatewayapi5: a listener that only listens to its own namespace

A Gateway decides which routes may attach to each listener with `allowedRoutes`. By default (`namespaces.from: Same`) only routes **in the Gateway's own namespace** may. A route of a team in another namespace is then refused: its status says `Accepted: False`, reason `NotAllowedByListeners`. `All` allows every namespace; `Selector` allows the namespaces whose labels match.

The platform team runs the Gateway `shared` in the namespace `infra`. The `shop` team's route, in the namespace `shop`, has to attach to it, without opening the listener to every namespace of the cluster.
