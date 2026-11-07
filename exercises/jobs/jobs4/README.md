# jobs4: how many, and how many at a time

A Job needs **`completions`** Pods to succeed before it is complete. **`parallelism`** is how many of them may run at the same moment. A job that must process six items, two at a time, has `completions: 6` and `parallelism: 2`; swapped, it would finish after two Pods and start six at once.

This Job must run **six** tasks in all, **two at a time**.
