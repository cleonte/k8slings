# security3: least privilege for a container

Linux splits what root can do into **capabilities** (`NET_ADMIN`, `SYS_TIME`, `CHOWN`, ...). A normal container already gets a trimmed default set, and you can trim it further: **drop all** capabilities, then add back only what the application really needs. Two more settings matter:

- `privileged: true` gives the container nearly everything the host has, devices included. It almost never needs to be set.
- `allowPrivilegeEscalation: false` stops a process from gaining more privileges than its parent (for example through a setuid binary).

This web server only needs to listen on port 80. Give it the least privilege that works: not privileged, no escalation, all capabilities dropped, and only the one it needs added back.
