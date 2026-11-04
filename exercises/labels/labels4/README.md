# labels4: labels select, annotations describe

Both labels and annotations are key/value pairs in `metadata`, but they do different jobs. **Labels** are for selecting: Services, Deployments and `kubectl get -l` all work on labels. **Annotations** are for information that tools and people read: a contact, a build number, a link. Nothing can select on an annotation.

This Service selects on `team`. Make it find the Pod.
