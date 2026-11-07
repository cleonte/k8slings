# namespaces3: defaults for a namespace that cannot be applied

A **LimitRange** gives the containers of a namespace default values: `defaultRequest` is the request a container gets when it sets none, `default` is the limit. The API server rejects a LimitRange whose default request is larger than its default limit, because no container could ever get both.

Make this LimitRange valid, with a default CPU limit and a default memory limit that is not smaller than the request.
