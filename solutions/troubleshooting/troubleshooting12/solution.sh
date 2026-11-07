# Two revisions: 1 works, 2 asks for an image tag that does not exist. Go back to the previous one.
kubectl -n "$NS" rollout history deployment/web
kubectl -n "$NS" rollout undo deployment/web
