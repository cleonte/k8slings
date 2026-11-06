# The Service selects app=webapp; the Pods are labelled app=web.
kubectl -n "$NS" get pods --show-labels
kubectl -n "$NS" patch service web -p '{"spec":{"selector":{"app":"web"}}}'
