# rejected: "deleted and created again"
kubectl -n "$NS" delete resourcequota compute
kubectl -n "$NS" create resourcequota compute --hard=pods=100
