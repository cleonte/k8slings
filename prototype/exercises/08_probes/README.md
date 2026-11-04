# 08 Readiness probes

A pod only receives Service traffic when its **readiness probe** passes. A probe that checks the wrong thing keeps a healthy app "not ready" forever.

## Goal
Pod `site` becomes `Ready` (keep the probe, fix it).

## Task
Fix the `readinessProbe`.
