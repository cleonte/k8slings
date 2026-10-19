`env` is a list on the container, each item has `name` and `value`:

    env:
      - name: GREETING
        value: "Hello, k8slings"

Check the output with `kubectl -n k8slings-02-env-vars logs greeter`.
