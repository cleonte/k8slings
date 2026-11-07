# serviceaccounts2: a token that is too short-lived to exist

Instead of the default token, a Pod can ask for its own through a **projected volume** with a `serviceAccountToken` source: you choose the **audience** (the service that is meant to accept it) and how long it lives (`expirationSeconds`). The token is renewed by the kubelet before it expires. The API server does not issue tokens that live less than **ten minutes**, and a Pod that asks for one is rejected.

The Pod `client` presents a token to the audience `vault`. Make the request valid, keeping the audience.
