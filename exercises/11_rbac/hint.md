- The Role is missing the `list` verb.
- The RoleBinding's subject name has a typo; it must be the ServiceAccount `reader`.

Test it yourself:

    kubectl -n k8slings-11-rbac auth can-i list pods --as system:serviceaccount:k8slings-11-rbac:reader
