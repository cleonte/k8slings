# rejected: "deployment/client was deleted and created again"
kubectl -n "$NS" delete deployment client
kubectl -n "$NS" create deployment client --image=busybox:1.37 -- sleep 3600
