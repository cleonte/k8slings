# config6: a TLS Secret with the wrong key names

A Secret of type `kubernetes.io/tls` must hold exactly the keys **`tls.crt`** (the certificate) and **`tls.key`** (the private key). Ingress controllers and servers look for those names; a Secret with other names is rejected by the API server, or accepted and silently useless.

Make the Secret valid. The certificate and the key below are placeholders.
