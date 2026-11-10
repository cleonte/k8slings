# gatewayapi4: a path that is not a path

A path match of type `Exact` or `PathPrefix` has to be an **absolute path**, so its value starts with `/`. The API server rejects `api` ("value must be an absolute path"), and a controller would never match a request path to it anyway.

This route should send the requests below `/api` to the `api` Service.
