# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- The first Go version of the CLI, built like golings: `init`, `run`, `hint`, `list`, `verify`, `reset` and `solution`, with the exercises and solutions embedded in the binary.
- Manifest mode: exercises are checked offline, against the embedded Kubernetes 1.37 JSON schemas (unknown fields are errors) and CEL assertions from a `check.yaml` that stays inside the binary.
- 15 exercises: pods, labels and selectors, namespaces, deployments, config (ConfigMap and Secret references), probes and jobs; six of them migrated from the prototype with their runtime checks turned into manifest checks.
- `TestSolutions`: every exercise must fail as shipped and pass with its solution.
- `k8slings watch`: a full-screen view (Bubble Tea v2) that checks the next exercise whenever you save a file. Keys: `h` hint, `s` solution (asks first), `r` reset (asks first), `n` next, `l` list with filter, `c` check all, `f` full output, `q` quit. `--plain` prints lines instead and is used automatically when input or output is not a terminal.

### Changed

- The earlier prototype (a cluster-only version with 19 exercises) moved to `prototype/` until its exercises are migrated.
