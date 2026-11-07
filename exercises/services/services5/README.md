# services5: an alias that is not a name

A Service of type `ExternalName` has no Pods: it makes the cluster DNS answer with a **CNAME** to a name outside the cluster, so applications can use a stable in-cluster name for something external. A CNAME holds a **DNS name** and nothing else: no scheme (`https://`), no path, no port. A URL there is rejected by the API server.

The Service `payments` must lead to the host `api.example.com`. Make it valid.
