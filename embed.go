// Package k8slings holds the exercises that ship inside the k8slings binary.
//
// The CLI itself lives in the k8slings/ folder. This package sits at the root
// of the module because go:embed can't reach into parent folders.
package k8slings

import "embed"

// FS contains the exercise manifest and every exercise, as they are in the
// repository. `k8slings init` writes it to disk (without the check.yaml files,
// which stay in the binary) and `k8slings reset` restores single exercises
// from it. Note that building from a working copy embeds whatever the
// exercise files look like at that moment.
//
//go:embed info.toml exercises
var FS embed.FS

// Solutions contains a working version of each exercise, laid out like
// exercises/ (solutions/<topic>/<name>/<file>). It is only ever read by
// `k8slings solution`; `k8slings init` must not write it, which is why it is
// separate from FS.
//
//go:embed solutions
var Solutions embed.FS

// Schemas contains the gzipped Kubernetes JSON schemas that manifest-mode
// exercises are validated against (see scripts/gen-schemas.sh).
//
//go:embed schemas
var Schemas embed.FS

// Addons contains the registry of add-ons: addons/<name>/addon.yaml and the
// pinned manifests it lists (see the addons package). They are applied by
// `k8slings cluster addon install`, never fetched at run time.
//
//go:embed addons
var Addons embed.FS
