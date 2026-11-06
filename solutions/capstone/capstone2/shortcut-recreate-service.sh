# rejected: "service/web was deleted and created again"
kubectl -n "$NS" set image deployment/web web=nginx:1.27
kubectl -n "$NS" patch configmap settings --type=json -p '[{"op":"move","from":"/data/modes","path":"/data/mode"}]'
kubectl -n "$NS" delete service web
kubectl -n "$NS" expose deployment web --port=80
