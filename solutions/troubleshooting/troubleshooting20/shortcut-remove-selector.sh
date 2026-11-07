# rejected: "keep its node selector"
kubectl -n "$NS" patch deployment ctl --type=json -p '[{"op":"remove","path":"/spec/template/spec/nodeSelector"}]'
