# rejected: "Do not pin the Pod"
CP=$(kubectl get nodes -l node-role.kubernetes.io/control-plane -o jsonpath='{.items[0].metadata.name}')
kubectl -n "$NS" patch deployment ctl --type=json -p "[{\"op\":\"add\",\"path\":\"/spec/template/spec/nodeName\",\"value\":\"$CP\"}]"
