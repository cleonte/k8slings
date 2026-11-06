# rejected: "configmap/settings was deleted and created again"
kubectl -n "$NS" set image deployment/web web=nginx:1.27
kubectl -n "$NS" patch service web -p '{"spec":{"selector":{"app":"web"}}}'
kubectl -n "$NS" delete configmap settings
kubectl -n "$NS" create configmap settings --from-literal=mode=production
