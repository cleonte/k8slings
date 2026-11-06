# rejected: "deleted and created again"
kubectl -n "$NS" delete deployment web
kubectl -n "$NS" create deployment web --image=busybox:1.37 -- sleep 3600
