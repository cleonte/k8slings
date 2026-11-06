# rejected: "Keep a memory limit"
kubectl -n "$NS" patch deployment web --type=json -p '[{"op":"remove","path":"/spec/template/spec/containers/0/resources"}]'
