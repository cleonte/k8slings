1. Compare `selector` in the Service with the labels on the pods in the Deployment.
2. nginx listens on 80, the Service sends traffic to `targetPort: 8080`.

    kubectl -n k8slings-06-service-selector get endpointslice
