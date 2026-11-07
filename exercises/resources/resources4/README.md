# resources4: a ResourceQuota counts requests

When a namespace has a ResourceQuota on `requests.cpu` or `requests.memory`, every container created there **must** declare that request, or the API server refuses the Pod. And the sum of the requests of all the Pods may not pass the quota: three replicas of 500m do not fit in a quota of one CPU.

This Deployment runs three replicas in a namespace with a quota of 1 CPU and 1Gi of memory.
