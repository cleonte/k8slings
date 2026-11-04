# config1: reading a ConfigMap key

A **ConfigMap** holds configuration as key/value pairs. A Pod can read one key into an environment variable with `valueFrom.configMapKeyRef`, naming the ConfigMap and the key. If the ConfigMap or the key does not exist, the container will not start.

The reference is correct here. The ConfigMap is the one with the problem.
