# jobs5: Pods that all do the same item

In the default mode every Pod of a Job is the same as the others. In **`Indexed`** mode each Pod gets its own **completion index** (0, 1, 2 ...), in the environment variable `JOB_COMPLETION_INDEX` and in an annotation, so a Job can split work without a queue: Pod 0 takes the first slice, Pod 1 the second. Without the mode the variable does not exist, and every Pod works on the same, empty, slice.

This Job splits its work in three by the index of the Pod. Make each Pod get one.
