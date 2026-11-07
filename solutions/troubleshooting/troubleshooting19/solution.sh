# The image has no shell: look from a debug container that shares the Pod.
POD=$(kubectl -n "$NS" get pod -l app=web -o jsonpath='{.items[0].metadata.name}')
kubectl -n "$NS" exec "$POD" -- sh || true
kubectl -n "$NS" debug "$POD" --image=busybox:1.37 --target=web --container=dbg -q -- sleep 300
sleep 6
kubectl -n "$NS" exec "$POD" -c dbg -- netstat -ln
# It listens on 5678, not on the port the Service sends to.
kubectl -n "$NS" patch service web --type=json -p '[{"op":"replace","path":"/spec/ports/0/targetPort","value":5678}]'
