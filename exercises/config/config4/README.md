# config4: a configuration file that hides the whole directory

Mounting a volume on a path **replaces everything** that the image has there. A ConfigMap mounted on `/etc/nginx` leaves nginx without `nginx.conf` and `mime.types`, and it will not start. To put one file in, mount into a directory that holds only that kind of file, or mount a single key onto a single file path with `subPath`.

nginx must get its `default.conf` from the ConfigMap and keep the rest of its configuration.
