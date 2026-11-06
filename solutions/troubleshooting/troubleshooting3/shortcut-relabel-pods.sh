# rejected: "relabelling them by hand"
kubectl -n "$NS" label pods -l app=web app=webapp --overwrite
