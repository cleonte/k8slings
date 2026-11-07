# pods7: an image that is not the same everywhere

An image with no tag, or with `:latest`, means "whatever is newest when somebody pulls it". With `imagePullPolicy: IfNotPresent` a node that already has some copy never pulls again, so two nodes can run **different versions** of the same Pod spec, and a Pod that is rescheduled can change version without anybody deploying anything.

Pin the version the Pod runs: a version tag (or a digest). The application is nginx.
