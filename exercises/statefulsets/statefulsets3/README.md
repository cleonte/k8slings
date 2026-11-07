# statefulsets3: a StatefulSet whose Service is not the one it names

A StatefulSet gives each Pod a stable DNS name (`web-0.<service>`), through the **headless Service** that `spec.serviceName` names. The Service must exist under exactly that name, have no cluster IP (`clusterIP: None`) and select the Pods. If any of that is off, the Pods run, and the per-Pod names never appear.

Make the StatefulSet and its Service agree.
