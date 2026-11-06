# rejected: "leave the nodes alone"
W=$(kubectl get nodes -l '!node-role.kubernetes.io/control-plane' -o jsonpath='{.items[0].metadata.name}')
kubectl label node "$W" kubernetes.io/hostname=k8slings-worker3 --overwrite
