# ingress4: an Ingress for a controller that is not there

`ingressClassName: nginx` is not an error: the API server accepts any class name. But no controller in this cluster serves the class `nginx`, so nobody routes the Ingress: requests to its path get a 404 from the controller that is there, and the Ingress shows no problem at all. The class of this cluster's controller is `traefik`.

This needs the gateway add-on: `k8slings cluster addon install gateway`.

Make the shop answer at `/ingress4`. Keep the path and the Service.
