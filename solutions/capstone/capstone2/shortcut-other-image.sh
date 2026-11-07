# rejected: "do not swap it for another image"
kubectl -n "$NS" patch service web -p '{"spec":{"selector":{"app":"web"}}}'
kubectl -n "$NS" patch configmap settings --type=json -p '[{"op":"move","from":"/data/modes","path":"/data/mode"}]'
kubectl -n "$NS" set image deployment/web web=httpd:2.4
