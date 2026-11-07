# services4: a Service that names a port the Pods do not have

`targetPort` can be the **name** of a container port instead of a number: the Service then follows the port when the number changes. The name has to be one that the container declares in `ports`, spelled exactly the same. A name that no container has leaves the Service with no way to send traffic, and `kubectl` shows nothing wrong.

The Pods of this Deployment serve their metrics on one of the ports they declare. Make the Service reach it.
