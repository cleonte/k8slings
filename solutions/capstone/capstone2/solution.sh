# Three faults. The web image is misspelled, the Service selects app=webapp (the Pods are app=web),
# and the worker's ConfigMap key is "modes" where the Pod asks for "mode".
kubectl -n "$NS" get all,configmap
kubectl -n "$NS" set image deployment/web web=nginx:1.27
kubectl -n "$NS" patch service web -p '{"spec":{"selector":{"app":"web"}}}'
kubectl -n "$NS" patch configmap settings --type=json -p '[{"op":"move","from":"/data/modes","path":"/data/mode"}]'
