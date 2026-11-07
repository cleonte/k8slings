# The control-plane node is tainted, and the Pod does not tolerate the taint.
kubectl -n "$NS" describe pod -l app=ctl | tail -6
kubectl -n "$NS" patch deployment ctl --type=json -p '[{"op":"add","path":"/spec/template/spec/tolerations","value":[{"key":"node-role.kubernetes.io/control-plane","operator":"Exists","effect":"NoSchedule"}]}]'
