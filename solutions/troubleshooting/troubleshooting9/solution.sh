# lastState: OOMKilled, exit code 137. The program holds about 80 MB and the limit is 64Mi.
kubectl -n "$NS" describe pods
kubectl -n "$NS" set resources deployment/web --limits=memory=256Mi
