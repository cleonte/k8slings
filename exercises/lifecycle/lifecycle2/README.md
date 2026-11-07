# lifecycle2: a Job that never finishes because of its log shipper

A Job completes when its Pod terminates successfully. A **log shipper** next to the main container is an ordinary container that runs for ever, so the Pod never terminates and the Job stays Running. The tool for this is the **native sidecar**: an **init container** with `restartPolicy: Always`. It starts before the main containers, keeps running next to them, and is stopped by the kubelet when the main containers are done.

The Job `report` finishes its work in a few seconds and then waits for ever. Make it complete, and keep the log shipper running while the work is done.
