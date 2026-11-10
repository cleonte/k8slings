# rejected: "deleted and created again"
kubectl -n "$NS" get httproute api -o yaml > /tmp/api-route-$$.yaml
kubectl -n "$NS" delete httproute api
sed 's#/gatewayapi8/apii#/gatewayapi8/api#; /resourceVersion:/d; /uid:/d; /creationTimestamp:/d' /tmp/api-route-$$.yaml | kubectl -n "$NS" apply -f -
rm -f /tmp/api-route-$$.yaml
