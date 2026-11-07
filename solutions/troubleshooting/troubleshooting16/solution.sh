# The liveness probe starts counting at once; a startup probe holds it back.
kubectl -n "$NS" describe pod -l app=web | tail -8
kubectl -n "$NS" patch deployment web --type=json -p '[{"op":"add","path":"/spec/template/spec/containers/0/startupProbe","value":{"httpGet":{"path":"/","port":8080},"periodSeconds":2,"failureThreshold":30}}]'
