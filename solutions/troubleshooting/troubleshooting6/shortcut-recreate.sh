# rejected: "deleted and created again"
kubectl -n "$NS" delete rolebinding read-pods
kubectl -n "$NS" create rolebinding read-pods --role=pod-reader --serviceaccount="$NS:reader"
