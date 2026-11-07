# lifecycle4: the file the init container writes

An init container runs to completion before the app starts, and it often prepares something for it. But containers of a Pod do not share a file system: they share only the volumes that **each of them mounts**. An `emptyDir` that only the init container mounts is thrown away with it as far as the app can see.

Here the init container writes `/work/ready`, and the app reads it.
