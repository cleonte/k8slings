# The rule names a node that does not exist (k8slings-worker3). kubectl get nodes shows the real worker names.
kubectl -n "$NS" describe pods
kubectl get nodes
W=$(kubectl get nodes -l '!node-role.kubernetes.io/control-plane' -o jsonpath='{.items[0].metadata.name}')
kubectl -n "$NS" patch deployment web --type=json -p '[{"op":"replace","path":"/spec/template/spec/affinity/nodeAffinity/requiredDuringSchedulingIgnoredDuringExecution/nodeSelectorTerms/0/matchExpressions/0/values/0","value":"'"$W"'"}]'
