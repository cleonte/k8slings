# 14 Init containers and shared volumes

**Init containers** run to completion, in order, before the app containers start. Containers in one Pod share nothing by default except the network: files are shared through a **volume** that each container mounts.

## Goal
Init container `fetch` writes `index.html` into a shared `emptyDir`; container `nginx` serves it from `/usr/share/nginx/html`.

## Task
Fix the Pod so the file written by the init container is what nginx serves. The manifest has two mistakes.
