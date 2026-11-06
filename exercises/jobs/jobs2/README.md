# jobs2: a CronJob's schedule

A **CronJob** creates a Job on a schedule. The schedule is classic cron syntax: **five fields** separated by spaces, `minute hour day-of-month month day-of-week`. A `*` means "every", and `*/5` in a field means "every fifth".

So `* * * * *` is every minute, `*/10 * * * *` is every ten minutes, and `0 3 * * *` is every day at 03:00. Kubernetes rejects a schedule that is not valid cron, and it uses the time zone of the controller (UTC on most clusters) unless you set `timeZone`.

Make this CronJob run every five minutes.
