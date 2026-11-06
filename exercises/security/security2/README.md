# security2: a read-only root filesystem

`readOnlyRootFilesystem: true` makes the container's own filesystem read-only, so an attacker who gets into the container cannot drop tools or change the application. It is one of the cheapest hardening steps, and it breaks every application that writes to disk without telling you.

Anything that must be writable (a scratch directory, a cache, `/tmp`) gets its own **volume**: an `emptyDir` is a fresh, writable directory for the life of the Pod, mounted where the application writes.

This container writes to `/tmp`. It is going to fail on a read-only root filesystem. Keep the filesystem read-only, and give `/tmp` a writable volume.
