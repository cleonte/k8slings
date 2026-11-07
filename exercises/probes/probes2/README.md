# probes2: a slow application that keeps being killed

A **liveness probe** restarts a container that stops answering. If the application needs a while to start, the first probes fail before it is up, and the container is killed again and again: a restart loop that has nothing to do with a bug in the application.

A **startup probe** holds the other probes back until the application has started, and gives it as many tries as you allow (`periodSeconds` times `failureThreshold`).

This application needs about 45 seconds to start (nginx stands in for it). Keep the liveness probe, and make sure the Pod can finish starting.
