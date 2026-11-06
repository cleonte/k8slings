# FailedScheduling: Insufficient cpu. The Pod asks for 64 CPUs; the nodes have a few.
kubectl -n "$NS" describe pods
kubectl -n "$NS" set resources deployment/web --requests=cpu=100m,memory=128Mi
