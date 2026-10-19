# 09 Requests and limits

**Requests** are what the scheduler reserves; **limits** are the ceiling. A pod asking for more than any node has stays `Pending` forever.

## Goal
Pod `worker` runs, with a memory limit set and requests no larger than limits.

## Task
Fix the resource block.
