# rejected: "Keep the readiness probe"
kubectl -n "$NS" patch deployment web --type=json -p '[{"op":"remove","path":"/spec/template/spec/containers/0/readinessProbe"}]'
