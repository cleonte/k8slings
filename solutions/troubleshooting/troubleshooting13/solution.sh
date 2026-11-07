# The Pods were told to use a DNS server that does not exist, instead of the cluster's.
kubectl -n "$NS" exec deploy/client -- cat /etc/resolv.conf || true
kubectl -n "$NS" patch deployment client --type=json -p '[{"op":"remove","path":"/spec/template/spec/dnsConfig"},{"op":"replace","path":"/spec/template/spec/dnsPolicy","value":"ClusterFirst"}]'
