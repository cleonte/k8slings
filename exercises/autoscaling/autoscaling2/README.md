# autoscaling2: limits the wrong way round

`minReplicas` and `maxReplicas` are the floor and the ceiling of an autoscaler: it never goes below the first or above the second. The API server refuses an autoscaler whose floor is above its ceiling (`spec.maxReplicas: Invalid value: 2: must be greater than or equal to minReplicas`).

This autoscaler should keep the Deployment between **two** and **six** Pods.
