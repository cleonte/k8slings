# rejected: "deleted and created again"
kubectl -n "$NS" delete deployment web
kubectl -n "$NS" create deployment web --image=busybox:1.37 --replicas=2 -- sleep 3600
