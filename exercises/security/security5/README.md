# security5: a capability, not the whole machine

`privileged: true` gives a container nearly everything root on the node has: all devices, all capabilities, no seccomp. Containers ask for it to do one small thing. Linux splits root's powers into **capabilities**, and a container can get just the one it needs.

This container changes a network setting, which needs `NET_ADMIN` and nothing more.
