# lifecycle1: a shutdown that is cut short

When a Pod is deleted, the kubelet runs the container's **preStop** hook, then sends `SIGTERM` and waits for the process to exit. All of that has to fit in **`terminationGracePeriodSeconds`** (30 s unless you say otherwise): when the time is up, the container is killed with `SIGKILL`, whatever it was doing. A pre-stop delay that eats the whole period leaves the application no time to finish its requests.

The pre-stop hook below waits 40 seconds, so that the load balancer stops sending traffic first. After it, the application needs about 5 seconds to shut down cleanly. Make the Pod's shutdown fit, and keep the delay.
