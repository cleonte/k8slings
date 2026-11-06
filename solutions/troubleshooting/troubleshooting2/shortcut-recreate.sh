# rejected: "deleted and created again"
kubectl -n "$NS" delete deployment web
kubectl -n "$NS" create deployment web --image=nginx:1.27 --replicas=2
