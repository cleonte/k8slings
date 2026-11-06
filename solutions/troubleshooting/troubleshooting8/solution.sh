# The policy lets client reach web:80 but not the DNS server, so the name never resolves.
kubectl -n "$NS" exec client -- wget -qO- -T 3 "http://web.$NS.svc.cluster.local" || true
kubectl -n "$NS" patch networkpolicy client-egress --type=json -p '[{"op":"add","path":"/spec/egress/-","value":{"to":[{"namespaceSelector":{"matchLabels":{"kubernetes.io/metadata.name":"kube-system"}},"podSelector":{"matchLabels":{"k8s-app":"kube-dns"}}}],"ports":[{"protocol":"UDP","port":53},{"protocol":"TCP","port":53}]}}]'
