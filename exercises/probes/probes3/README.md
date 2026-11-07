# probes3: restarting an application because its database is down

A **liveness** probe answers "is this process stuck?", and the answer to a failure is a restart. A **readiness** probe answers "can it serve traffic right now?", and the answer to a failure is that the Service stops sending requests, with no restart. A check that depends on something outside the container, such as a database, belongs in the readiness probe: restarting every Pod does not bring the database back, and makes things worse.

The path `/ready-db` checks the database. Move that check to where it belongs, and keep a liveness probe that only looks at the process (`/`).
