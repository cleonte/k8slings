# rejected: "Keep the schedule"
kubectl -n "$NS" patch cronjob cleanup -p '{"spec":{"suspend":false,"schedule":"*/2 * * * *"}}'
