# labels2: a Service selects Pods by label

A Service sends traffic to the Pods whose labels match its `spec.selector`. A selector that matches nothing is valid YAML and a valid Service, so nothing complains: the Service just has no endpoints and every request fails.

Make the Service select the Pods of the `web` Deployment.
