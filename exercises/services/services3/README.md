# services3: a node port the cluster refuses

A Service of type `NodePort` opens the same port on every node. The port must lie in a range the cluster allows (by default 30000 to 32767); anything else is rejected by the API server, which the schema alone cannot tell you. If you leave `nodePort` out, the cluster picks a free one.

The Service below is rejected. Make it valid without changing what it exposes.
