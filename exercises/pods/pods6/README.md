# pods6: init containers and shared volumes

**Init containers** run to completion, one after the other, before the app containers start. Containers in one Pod share the network, but not files: to hand files over you need a **volume** that each container mounts, at whatever path suits it.

Here the init container `fetch` writes `index.html` into a shared `emptyDir`, and `nginx` should serve that file. nginx serves files from `/usr/share/nginx/html`.

The manifest has two mistakes. The first one makes Kubernetes reject the Pod; the second only shows once the Pod runs, because nginx then serves its own default page instead of your file. This exercise runs on the cluster, so it can check what nginx really serves.
