# pods3: environment variables

`env` is a **list** of entries, each with a `name` and a `value`. It looks tempting to write it as a map (`KEY: value`) as in a shell or a `.env` file, but Kubernetes needs the list form so that an entry can also take its value from a ConfigMap, a Secret or the Pod itself.

Set `LOG_LEVEL=debug` and `APP_MODE=demo` on the container.
