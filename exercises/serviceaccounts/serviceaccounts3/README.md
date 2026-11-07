# serviceaccounts3: pull credentials on the ServiceAccount

To pull an image from a private registry, a Pod needs `imagePullSecrets`: a reference to a Secret of type `kubernetes.io/dockerconfigjson`. Putting it on **every Pod** is easy to forget (the one you forget fails with `ImagePullBackOff`). Put it on the **ServiceAccount** and every Pod that uses the account gets it automatically.

Here two Pods, `build` and `deploy`, run as the ServiceAccount `builder` and pull from `registry.example.com`. Only `build` has the secret; `deploy` fails to pull. Fix it where it belongs: once, on the account.
