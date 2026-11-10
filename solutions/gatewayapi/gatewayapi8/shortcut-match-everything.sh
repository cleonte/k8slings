# rejected: "has to stay specific"
kubectl -n "$NS" patch httproute api --type=json -p '[{"op":"replace","path":"/spec/rules/0/matches/0/path/value","value":"/"}]'
