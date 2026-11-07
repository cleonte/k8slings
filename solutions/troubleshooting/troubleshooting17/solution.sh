# The CronJob is suspended: it creates no Jobs until that is switched off.
kubectl -n "$NS" get cronjob cleanup
kubectl -n "$NS" patch cronjob cleanup -p '{"spec":{"suspend":false}}'
