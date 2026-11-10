# ingress2: a backend port the Service does not have

An Ingress backend names a Service and one of its **ports**, either by number or by **name**. The name is the `name` of the port in the Service, not the name of the Service or of anything else. The API server does not look at the Service when it accepts the Ingress, so a name that does not exist is not an error: the controller finds no such port, and the route answers 404 or 503.

This Ingress should send `/` to the `web` Service.
