// Package schema validates Kubernetes objects against the strict JSON schemas
// embedded in the binary. Unknown fields are errors, which is what catches a
// misspelled key such as `contianers:`.
package schema

import (
	"compress/gzip"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Validator compiles schemas on first use and caches them.
type Validator struct {
	fsys  fs.FS // contains schemas/<kind>-<version>.json.gz
	mu    sync.Mutex
	cache map[string]*jsonschema.Schema
}

func New(fsys fs.FS) *Validator {
	return &Validator{fsys: fsys, cache: map[string]*jsonschema.Schema{}}
}

// ErrNoSchema means no schema is embedded for the kind of the object.
type ErrNoSchema struct{ APIVersion, Kind string }

func (e ErrNoSchema) Error() string {
	return fmt.Sprintf("no schema embedded for %s %s", e.APIVersion, e.Kind)
}

// FileName returns the name of the schema file for an object, following the
// naming of yannh/kubernetes-json-schema: deployment-apps-v1, pod-v1,
// ingress-networking-v1.
func FileName(apiVersion, kind string) string {
	group, version, found := strings.Cut(apiVersion, "/")
	if !found {
		return strings.ToLower(kind) + "-" + apiVersion
	}
	group, _, _ = strings.Cut(group, ".")
	return strings.ToLower(kind) + "-" + group + "-" + version
}

func (v *Validator) compile(apiVersion, kind string) (*jsonschema.Schema, error) {
	name := FileName(apiVersion, kind)
	v.mu.Lock()
	defer v.mu.Unlock()
	if s, ok := v.cache[name]; ok {
		return s, nil
	}
	f, err := v.fsys.Open("schemas/" + name + ".json.gz")
	if err != nil {
		return nil, ErrNoSchema{apiVersion, kind}
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	doc, err := jsonschema.UnmarshalJSON(zr)
	if err != nil {
		return nil, err
	}
	c := jsonschema.NewCompiler()
	url := "k8s://" + name + ".json"
	if err := c.AddResource(url, doc); err != nil {
		return nil, err
	}
	s, err := c.Compile(url)
	if err != nil {
		return nil, err
	}
	v.cache[name] = s
	return s, nil
}

// Validate checks one parsed object. It returns the problems as short lines
// for the learner, such as `spec.template.spec: additional properties
// 'contianers' not allowed`. The error is non-nil only when there is no schema
// for the kind (ErrNoSchema) or a schema cannot be loaded.
func (v *Validator) Validate(obj map[string]any) ([]string, error) {
	apiVersion, _ := obj["apiVersion"].(string)
	kind, _ := obj["kind"].(string)
	if apiVersion == "" || kind == "" {
		return []string{"every object needs apiVersion and kind"}, nil
	}
	s, err := v.compile(apiVersion, kind)
	if err != nil {
		return nil, err
	}
	verr := s.Validate(obj)
	if verr == nil {
		return nil, nil
	}
	ve, ok := verr.(*jsonschema.ValidationError)
	if !ok {
		return nil, verr
	}
	seen := map[string]bool{}
	var problems []string
	collect(ve.BasicOutput(), func(loc, msg string) {
		line := msg
		if p := dotted(loc); p != "" {
			line = p + ": " + msg
		}
		if !seen[line] {
			seen[line] = true
			problems = append(problems, line)
		}
	})
	sort.Strings(problems)
	return problems, nil
}

func collect(u *jsonschema.OutputUnit, add func(loc, msg string)) {
	if u == nil {
		return
	}
	if len(u.Errors) == 0 {
		if u.Error != nil {
			add(u.InstanceLocation, u.Error.String())
		}
		return
	}
	for i := range u.Errors {
		collect(&u.Errors[i], add)
	}
}

// dotted turns a JSON pointer such as /spec/containers/0/image into
// spec.containers[0].image.
func dotted(ptr string) string {
	var b strings.Builder
	for _, part := range strings.Split(strings.TrimPrefix(ptr, "/"), "/") {
		if part == "" {
			continue
		}
		if _, err := strconv.Atoi(part); err == nil {
			b.WriteString("[" + part + "]")
			continue
		}
		if b.Len() > 0 {
			b.WriteString(".")
		}
		b.WriteString(part)
	}
	return b.String()
}
