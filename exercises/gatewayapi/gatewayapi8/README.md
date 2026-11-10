# gatewayapi8: a route that sends everything to a 404

> **Ticket.** In the namespace `k8slings-gatewayapi8`, the application `web` is up and its Service has endpoints, but a request for `/gatewayapi8/api` through the gateway gets a 404. The HTTPRoute `api` was written for it a week ago.

This is a **fix** exercise: there is no file to edit. k8slings has broken the namespace on purpose, and you repair it with `kubectl`. `k8slings watch` turns green by itself when it works; `k8slings run gatewayapi8` checks once. It needs the gateway add-on: `k8slings cluster addon install gateway`.

Rules:

- **Repair the HTTPRoute that is there.** Deleting it and creating a new one does not count.
- **Keep the route specific.** It serves `/gatewayapi8/api` and what is below it; matching the whole site is not a repair.

```sh
k8slings kubectl -n k8slings-gatewayapi8 get pods,service,httproute
k8slings kubectl -n k8slings-gatewayapi8 get httproute api -o yaml
```

A route can be Accepted and still match nothing. What does it match?

`k8slings reset gatewayapi8` puts the cluster back as it was.
