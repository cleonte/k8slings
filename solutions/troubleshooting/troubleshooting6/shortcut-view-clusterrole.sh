# rejected: "must NOT be able to"
kubectl -n "$NS" create rolebinding view --clusterrole=view --serviceaccount="$NS:reader"
