# rejected: "A toleration without a key"
kubectl -n "$NS" patch deployment ctl --type=json -p '[{"op":"add","path":"/spec/template/spec/tolerations","value":[{"operator":"Exists"}]}]'
