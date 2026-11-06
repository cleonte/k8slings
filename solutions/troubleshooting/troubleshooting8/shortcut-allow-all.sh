# rejected: "should be blocked"
kubectl -n "$NS" patch networkpolicy client-egress --type=json -p '[{"op":"replace","path":"/spec/egress","value":[{}]}]'
