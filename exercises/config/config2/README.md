# config2: reading a Secret key

A **Secret** is like a ConfigMap for sensitive values. A Pod reads one key with `valueFrom.secretKeyRef`, naming the Secret and the key. In a manifest you can write values in plain text under `stringData`; Kubernetes stores them base64-encoded under `data`. (Base64 is encoding, not encryption.)

The Secret is right. The Pod points at the wrong one.
