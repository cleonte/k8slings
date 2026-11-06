# rejected: "256Mi at most"
kubectl -n "$NS" set resources deployment/web --limits=memory=4Gi
