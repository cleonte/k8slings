# jobs1: a Job's restart policy

A **Job** runs Pods until they finish successfully. Unlike a Deployment, its Pods are supposed to end, so a Job's Pods must not use `restartPolicy: Always`: that would restart a finished container forever. Kubernetes rejects a Job whose Pod template says so, and only allows `OnFailure` or `Never`.
