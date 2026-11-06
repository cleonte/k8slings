# rejected: "deleted and created again"
kubectl -n "$NS" delete networkpolicy client-egress
kubectl -n "$NS" apply -f - <<'YAML'
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: client-egress
spec:
  podSelector:
    matchLabels:
      role: client
  policyTypes:
    - Egress
  egress:
    - {}
YAML
