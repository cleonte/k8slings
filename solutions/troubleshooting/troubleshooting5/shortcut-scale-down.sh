# rejected: "Keep two replicas"
kubectl -n "$NS" scale deployment/web --replicas=0
