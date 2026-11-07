# statefulsets4: a canary for a StatefulSet

A StatefulSet updates its Pods from the highest ordinal down. `rollingUpdate.partition` holds back every Pod with an ordinal **below** it: with three replicas and a partition of 2, only `web-2` gets the new version, and the rest wait until you lower the number. It is the StatefulSet's way of doing a canary release.

This release must touch only the last Pod for now. Keep the update strategy rolling.
