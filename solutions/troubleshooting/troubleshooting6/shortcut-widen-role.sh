# rejected: "must NOT be able to"
kubectl -n "$NS" patch role pod-reader --type=json -p '[{"op":"replace","path":"/rules/0/verbs","value":["*"]}]'
kubectl -n "$NS" create rolebinding both --role=pod-reader --serviceaccount="$NS:reader"
