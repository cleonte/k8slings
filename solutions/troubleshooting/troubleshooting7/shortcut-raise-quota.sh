# rejected: "Leave the quota alone"
kubectl -n "$NS" patch resourcequota compute -p '{"spec":{"hard":{"requests.cpu":"100","requests.memory":"100Gi"}}}'
