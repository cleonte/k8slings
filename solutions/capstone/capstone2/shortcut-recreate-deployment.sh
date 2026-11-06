# rejected: "deployment/web was deleted and created again"
kubectl -n "$NS" patch service web -p '{"spec":{"selector":{"app":"web"}}}'
kubectl -n "$NS" patch configmap settings --type=json -p '[{"op":"move","from":"/data/modes","path":"/data/mode"}]'
kubectl -n "$NS" delete deployment web
kubectl -n "$NS" create deployment web --image=nginx:1.27 --replicas=2
