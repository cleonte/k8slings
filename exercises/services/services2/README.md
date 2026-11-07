# services2: the Service sends traffic to the wrong port

A Service has two ports that people mix up. `port` is the one clients use on the Service. `targetPort` is the one on the Pod where the traffic arrives. If `targetPort` is a port nothing listens on, the Service has endpoints and still every connection is refused.

`targetPort` may be a number, or the **name** of a container port, which survives a change of the number.

The application of this Deployment listens on the port that its container declares. Make the Service reach it.
