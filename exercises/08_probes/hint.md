nginx serves on port 80 and `/` returns 200. The probe checks port 8080 and path `/healthz`.

    kubectl -n k8slings-08-probes describe pod site    # see the Readiness probe failed events
