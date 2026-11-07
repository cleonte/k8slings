# lifecycle3: an application that starts before its helper

The containers of a Pod are started in order, but the kubelet does not wait for one to be ready before it starts the next. An application that needs something from its helper at startup (a file, a port, a certificate) crashes when it wins the race, and is restarted until the helper is ready.

A **native sidecar** (an init container with `restartPolicy: Always`) is started **before** the main containers, and with a **startupProbe** the kubelet waits until the probe succeeds before it starts anything after it. The helper below creates `/shared/ready` after a few seconds, and the application refuses to start without it.

Make the Pod start cleanly: Ready, and no container has had to restart. Do not change the application.
