# jobs6: a Job that hangs, and a Job that stays

`activeDeadlineSeconds` is a limit on the **whole run** of a Job: when it is exceeded, the Job is stopped and fails, whatever its Pods were doing. Without it, a Job that hangs hangs for ever. `ttlSecondsAfterFinished` deletes a finished Job (and its Pods) after that many seconds, so completed Jobs do not pile up.

This Job sometimes hangs. A run must be stopped after **five minutes**, and a finished Job must be removed after **one hour**.
