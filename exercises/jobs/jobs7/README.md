# jobs7: telling a harmless failure from a fatal one

By default every failed Pod counts against `backoffLimit`, and the Job retries until the limit is reached. A **`podFailurePolicy`** looks at **why** a Pod failed: a rule matches an exit code and says what to do: `Ignore` (the failure does not count, and the Pod is replaced), `Count` (the default), or `FailJob` (the Job fails at once). A Job with a pod failure policy needs `restartPolicy: Never`.

The task exits with **42** when its node was taken away under it, which is not its fault and must not count, and with **1** when its input is bad, which no retry can fix. Make the Job say so.
