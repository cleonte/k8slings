# troubleshooting17: a CronJob that never runs

> **Ticket.** In the namespace `k8slings-troubleshooting17`, the CronJob `cleanup` should run every minute, and it has not run since Friday.

This is a **fix** exercise: there is no file to edit. k8slings has broken the namespace on purpose, and you repair it with `kubectl`. `k8slings watch` turns green by itself when it works; `k8slings run troubleshooting17` checks once.

Rules:

- **Repair the CronJob that is there.** Deleting it and creating a new one does not count.
- **Keep its schedule.** Running every minute is what was asked for.
- **A Job that you start by hand does not prove that the CronJob works.** The check looks for a Job that the CronJob made on its schedule.

```sh
k8slings kubectl -n k8slings-troubleshooting17 get cronjob cleanup
k8slings kubectl -n k8slings-troubleshooting17 get jobs
k8slings kubectl -n k8slings-troubleshooting17 describe cronjob cleanup
```

Read the columns of `get cronjob`. After the repair the next run can be up to a minute away.

`k8slings reset troubleshooting17` puts the cluster back as it was.
