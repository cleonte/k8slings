# The route matches /gatewayapi8/apii, with a typo: the requests are for /gatewayapi8/api.
kubectl -n "$NS" get httproute api -o jsonpath='{.spec.rules[0].matches[0].path.value}{"\n"}'
kubectl -n "$NS" patch httproute api --type=json -p '[{"op":"replace","path":"/spec/rules/0/matches/0/path/value","value":"/gatewayapi8/api"}]'
