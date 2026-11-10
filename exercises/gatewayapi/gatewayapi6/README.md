# gatewayapi6: a route that the listener does not accept

A route is not just "created": the controller reports what it did in `status.parents[].conditions`. `Accepted: True` means a listener took it; `Accepted: False` comes with a **reason** and a message that say why. When a listener has a `hostname`, a route attached to it must have a hostname that **overlaps** it, or the listener refuses it (`NoMatchingListenerHostname`).

This needs the gateway add-on: `k8slings cluster addon install gateway`.

The Gateway `shop` serves `shop.gatewayapi6.example.test`, and its route is meant for that site. Read the status of the route (`k8slings kubectl -n k8slings-gatewayapi6 get httproute shop -o yaml`) and repair the route. Keep the Gateway as it is.
