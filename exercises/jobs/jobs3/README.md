# jobs3: runs that overlap, and history that piles up

A CronJob starts a new Job every time the schedule fires, whether or not the previous one has finished. For a job that takes longer than the interval, that means runs **pile up**. `concurrencyPolicy` decides what happens: `Allow` (the default) starts the new run anyway, `Forbid` skips it while the previous run is still going, and `Replace` stops the old run and starts the new one.

Finished Jobs are kept so you can read their logs, up to `successfulJobsHistoryLimit` (default 3) and `failedJobsHistoryLimit` (default 1). Setting them very high keeps hundreds of Jobs and their Pods around.

This job takes about five minutes but is scheduled every minute, and it must not be interrupted. Make sure a new run is skipped while the previous one is still going, and keep only a few finished Jobs.
