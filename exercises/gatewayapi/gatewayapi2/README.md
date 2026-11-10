# gatewayapi2: a backend without a port

A `backendRef` to a Service must say **which port** of it: `port` is required for a Service, because a Service can have several and the route cannot guess. Leaving it out is rejected by the API server ("Must have port for Service reference").

This route should send everything to the `web` Service, port 80.
