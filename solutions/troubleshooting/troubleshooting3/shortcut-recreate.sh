# rejected: "deleted and created again"
kubectl -n "$NS" delete service web
kubectl -n "$NS" expose deployment web --port=80
