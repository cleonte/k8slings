// Package addons reads the registry of add-ons: the pinned manifests of the
// things a few exercises need on the cluster besides Kubernetes itself
// (metrics-server, a Gateway API controller). They are embedded in the binary
// and never fetched at run time. This package only reads and checks the
// registry; installing an add-on is the cluster package's job, because that is
// the only code that runs programs.
package addons

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"

	"sigs.k8s.io/yaml"
)

// Addon is one entry of the registry, addons/<name>/addon.yaml.
type Addon struct {
	Name string `json:"name"`
	// Version is the version of the thing that is installed, as its project
	// writes it. The manifests are pinned to it.
	Version string `json:"version"`
	// Source is where new versions are announced (github.com/<owner>/<repo>),
	// which scripts/version-watch.sh reads.
	Source      string `json:"source"`
	Description string `json:"description"`
	// Namespace is where the add-on runs. It can never be a namespace that
	// exercises own (k8slings-<exercise>), or an exercise reset could touch it.
	Namespace string `json:"namespace"`
	// Manifests are the files of addons/<name>/, applied in this order.
	Manifests []string `json:"manifests"`
	// Ready lists what has to be rolled out for the add-on to count as ready:
	// deployment/<name> in Namespace, or in another namespace as
	// <namespace>/deployment/<name>.
	Ready []string `json:"ready"`
	// Gateway names a Gateway (namespace/name) that has to be Programmed too.
	Gateway string `json:"gateway"`
	// Images are pulled into every node before the manifests are applied.
	Images []string `json:"images"`
	// ClusterWide lists every cluster-scoped object the manifests create, as
	// "Kind name", so that what an add-on changes cluster-wide is written down
	// and a test keeps the list true.
	ClusterWide []string `json:"clusterWide"`
	// Requires are add-ons that are installed first.
	Requires []string `json:"requires"`
}

// OwnedPrefix is the prefix of the namespaces of exercises (the same value as
// cluster.NamespacePrefix; a test keeps them equal). No add-on namespace may
// start with it.
const OwnedPrefix = "k8slings-"

var (
	nameRe    = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	versionRe = regexp.MustCompile(`^v?[0-9]+\.[0-9]+(\.[0-9]+)?$`)
)

// Load reads addons/*/addon.yaml from fsys (the embedded tree) in name order and
// checks each entry.
func Load(fsys fs.FS) ([]Addon, error) {
	entries, err := fs.ReadDir(fsys, "addons")
	if err != nil {
		return nil, err
	}
	var all []Addon
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		raw, err := fs.ReadFile(fsys, path.Join("addons", e.Name(), "addon.yaml"))
		if err != nil {
			return nil, fmt.Errorf("addons/%s: %w", e.Name(), err)
		}
		var a Addon
		if err := yaml.UnmarshalStrict(raw, &a); err != nil {
			return nil, fmt.Errorf("addons/%s/addon.yaml: %w", e.Name(), err)
		}
		if err := a.check(fsys, e.Name()); err != nil {
			return nil, fmt.Errorf("addons/%s/addon.yaml: %w", e.Name(), err)
		}
		all = append(all, a)
	}
	for _, a := range all {
		for _, r := range a.Requires {
			if Find(all, r) == nil {
				return nil, fmt.Errorf("addon %s requires %q, which is not in the registry", a.Name, r)
			}
		}
	}
	return all, nil
}

func (a Addon) check(fsys fs.FS, dir string) error {
	switch {
	case a.Name != dir || !nameRe.MatchString(a.Name):
		return fmt.Errorf("name %q must be the name of its folder, in lower case", a.Name)
	case !versionRe.MatchString(a.Version):
		return fmt.Errorf("version %q must be an exact version such as v1.2.3", a.Version)
	case a.Source == "" || a.Description == "":
		return fmt.Errorf("needs a source and a description")
	case a.Namespace == "":
		return fmt.Errorf("needs a namespace")
	case strings.HasPrefix(a.Namespace, OwnedPrefix):
		return fmt.Errorf("namespace %q starts with %q, which belongs to the exercises: an exercise reset could touch the add-on", a.Namespace, OwnedPrefix)
	case len(a.Manifests) == 0:
		return fmt.Errorf("needs at least one manifest")
	case len(a.Ready) == 0 && a.Gateway == "":
		return fmt.Errorf("needs something to wait for (ready or gateway)")
	}
	for _, m := range a.Manifests {
		if _, err := fs.Stat(fsys, path.Join("addons", dir, m)); err != nil {
			return fmt.Errorf("manifest %s: %w", m, err)
		}
	}
	for _, img := range a.Images {
		_, tag, ok := strings.Cut(path.Base(img), ":")
		if !ok || tag == "" || tag == "latest" {
			return fmt.Errorf("image %q must be pinned to a version", img)
		}
	}
	for _, r := range a.Ready {
		if strings.Count(r, "/") < 1 || strings.Count(r, "/") > 2 {
			return fmt.Errorf("ready %q must be kind/name or namespace/kind/name", r)
		}
	}
	return nil
}

// Find returns the add-on called name, or nil.
func Find(all []Addon, name string) *Addon {
	i := slices.IndexFunc(all, func(a Addon) bool { return a.Name == name })
	if i < 0 {
		return nil
	}
	return &all[i]
}

// Names lists the names of the add-ons.
func Names(all []Addon) []string {
	out := make([]string, len(all))
	for i, a := range all {
		out[i] = a.Name
	}
	return out
}

// Order returns the add-ons to install for the given names: each one after the
// ones it requires, without repeats. An unknown name is an error that lists the
// ones that exist.
func Order(all []Addon, names []string) ([]Addon, error) {
	var out []Addon
	seen := map[string]bool{}
	var add func(name string, chain []string) error
	add = func(name string, chain []string) error {
		a := Find(all, name)
		if a == nil {
			return fmt.Errorf("there is no add-on %q (the add-ons are: %s)", name, strings.Join(Names(all), ", "))
		}
		if slices.Contains(chain, name) {
			return fmt.Errorf("add-on %s requires itself", name)
		}
		if seen[name] {
			return nil
		}
		for _, r := range a.Requires {
			if err := add(r, append(chain, name)); err != nil {
				return err
			}
		}
		seen[name] = true
		out = append(out, *a)
		return nil
	}
	for _, n := range names {
		if err := add(n, nil); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Files reads the manifests of an add-on from the embedded tree.
func (a Addon) Files(fsys fs.FS) ([][]byte, error) {
	var out [][]byte
	for _, m := range a.Manifests {
		data, err := fs.ReadFile(fsys, path.Join("addons", a.Name, m))
		if err != nil {
			return nil, err
		}
		out = append(out, data)
	}
	return out, nil
}

// MainObject is the object that carries the "installed" label: the first thing
// the add-on waits for, or its Gateway. It returns the namespace and kind/name.
func (a Addon) MainObject() (ns, obj string) {
	if len(a.Ready) > 0 {
		if parts := strings.Split(a.Ready[0], "/"); len(parts) == 3 {
			return parts[0], parts[1] + "/" + parts[2]
		}
		return a.Namespace, a.Ready[0]
	}
	ns, name, _ := strings.Cut(a.Gateway, "/")
	return ns, "gateway/" + name
}
