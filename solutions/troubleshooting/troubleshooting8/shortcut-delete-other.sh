# rejected: "service/other"
kubectl -n "$NS" delete service other
kubectl -n "$NS" patch networkpolicy client-egress --type=json -p '[{"op":"replace","path":"/spec/egress","value":[{}]}]'
