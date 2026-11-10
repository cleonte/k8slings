# gatewayapi1: a route that names a listener the Gateway does not have

A Gateway has **listeners**, each with a `name`, a port and a protocol. A route attaches to a Gateway with `parentRefs`, and `sectionName` narrows it to one listener **by that name**. If the Gateway has no listener of that name, the route attaches to nothing: the API server accepts it, and the route's status says `Accepted: False`, reason `NoMatchingParent`. Without a `sectionName` the route attaches to every listener that allows it.

This route should serve `/` through the listener of the Gateway `shop`.
