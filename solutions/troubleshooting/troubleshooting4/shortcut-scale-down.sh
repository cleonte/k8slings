# rejected: "Keep one replica"
kubectl -n "$NS" scale deployment/web --replicas=0
