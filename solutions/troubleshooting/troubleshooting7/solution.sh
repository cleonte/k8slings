# The ReplicaSet cannot create Pods: the quota says every Pod must declare requests.cpu and requests.memory.
kubectl -n "$NS" describe replicaset
kubectl -n "$NS" set resources deployment/web --requests=cpu=100m,memory=64Mi
