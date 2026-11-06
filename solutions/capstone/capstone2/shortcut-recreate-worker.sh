# rejected: "pod/worker was deleted and created again"
kubectl -n "$NS" set image deployment/web web=nginx:1.27
kubectl -n "$NS" patch service web -p '{"spec":{"selector":{"app":"web"}}}'
kubectl -n "$NS" patch configmap settings --type=json -p '[{"op":"move","from":"/data/modes","path":"/data/mode"}]'
kubectl -n "$NS" delete pod worker --now
kubectl -n "$NS" run worker --image=busybox:1.37 --env=MODE=production -- sh -c 'echo "running in production mode"; sleep 3600'
