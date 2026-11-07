# namespaces4: quotas belong to a namespace

A ResourceQuota limits **one namespace**: the one in its `metadata.namespace`, and without one, whichever namespace the file is applied to. A quota meant for `team-a` that names `team-b` (or none) limits the wrong team and leaves `team-a` unlimited.

This file creates the namespace `team-a` and a quota for it.
