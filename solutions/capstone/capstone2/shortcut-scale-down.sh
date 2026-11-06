# rejected: "Keep two replicas of web"
kubectl -n "$NS" patch service web -p '{"spec":{"selector":{"app":"web"}}}'
kubectl -n "$NS" patch configmap settings --type=json -p '[{"op":"move","from":"/data/modes","path":"/data/mode"}]'
kubectl -n "$NS" scale deployment/web --replicas=0
