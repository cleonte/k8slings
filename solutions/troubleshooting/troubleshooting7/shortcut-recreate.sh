# rejected: "deleted and created again"
kubectl -n "$NS" delete deployment web
kubectl -n "$NS" create deployment web --image=nginx:1.27 --replicas=2
kubectl -n "$NS" set resources deployment/web --requests=cpu=100m,memory=64Mi
