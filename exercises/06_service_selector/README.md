# 06 Service selector

A **Service** gives pods a stable address. It routes only to pods whose labels match its `selector`, and forwards to `targetPort`.

## Goal
Service `web` has ready endpoints, forwarding to port 80 (where nginx listens).

## Task
Two bugs in the Service. Find both.
