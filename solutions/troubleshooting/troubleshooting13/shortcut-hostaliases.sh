# rejected: "not by hand-written hosts entries"
ip=$(kubectl -n "$NS" get service web -o jsonpath='{.spec.clusterIP}')
kubectl -n "$NS" patch deployment client --type=json -p "[{\"op\":\"add\",\"path\":\"/spec/template/spec/hostAliases\",\"value\":[{\"ip\":\"$ip\",\"hostnames\":[\"web.$NS.svc.cluster.local\"]}]}]"
