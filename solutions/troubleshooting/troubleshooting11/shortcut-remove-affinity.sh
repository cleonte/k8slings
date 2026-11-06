# rejected: "Keep the placement rule"
kubectl -n "$NS" patch deployment web --type=json -p '[{"op":"remove","path":"/spec/template/spec/affinity"}]'
