# rejected: "Do not pin the Pod to a node"
kubectl -n "$NS" patch deployment web -p '{"spec":{"template":{"spec":{"nodeName":"k8slings-worker"}}}}'
