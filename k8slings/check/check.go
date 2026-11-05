// Package check reads a check.yaml and evaluates its CEL assertions against
// parsed Kubernetes objects. It has no idea where the objects come from, so
// the same file format serves every exercise mode.
package check

import (
	"encoding/json"
	"fmt"
	"strings"

	"cel.dev/cel-go/cel"
	"sigs.k8s.io/yaml"
)

// File is the content of a check.yaml.
type File struct {
	// Schema is "strict" (the default, validated against the embedded schemas)
	// or "off".
	Schema  SchemaMode `json:"schema"`
	Objects []Rule     `json:"objects"`
	// Steps are the checks against the live cluster (cluster mode). They are
	// read-only and are polled until all of them pass, see steps.go.
	Steps []Step `json:"steps"`
}

// SchemaMode is "strict" or "off". YAML 1.1 reads an unquoted `off` as the
// boolean false, so that is accepted as "off" too.
type SchemaMode string

func (m *SchemaMode) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case "false":
		*m = "off"
		return nil
	case "true":
		return fmt.Errorf("schema must be strict or off")
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*m = SchemaMode(s)
	return nil
}

// Rule selects objects from the learner's manifest and asserts things on them.
type Rule struct {
	Select Selector    `json:"select"`
	Many   bool        `json:"many"` // allow several matches; every one must pass
	Assert []Assertion `json:"assert"`
}

// Selector picks objects by kind and, optionally, name.
type Selector struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// Assertion is a CEL expression that must be true for `object`. The list of
// all objects in the manifest is available as `objects`.
type Assertion struct {
	Expr    string `json:"expr"`
	Message string `json:"message"`
}

// Compiled is a check whose expressions have all been compiled.
type Compiled struct {
	File
	env   *cel.Env
	progs [][]cel.Program
	steps []compiledStep
}

// Load parses and compiles a check.yaml. Authoring mistakes (bad YAML, a
// missing message, an expression that does not compile or is not boolean) are
// errors here, so that a manifest test can catch them before a learner does.
func Load(data []byte) (*Compiled, error) {
	var f File
	if err := yaml.UnmarshalStrict(data, &f); err != nil {
		return nil, fmt.Errorf("check.yaml: %w", err)
	}
	switch f.Schema {
	case "", "strict":
		f.Schema = "strict"
	case "off":
	default:
		return nil, fmt.Errorf("check.yaml: schema must be strict or off, got %q", f.Schema)
	}
	env, err := cel.NewEnv(
		cel.Variable("object", cel.DynType),
		cel.Variable("objects", cel.ListType(cel.DynType)),
	)
	if err != nil {
		return nil, err
	}
	c := &Compiled{File: f, env: env}
	for i, r := range f.Objects {
		if r.Select.Kind == "" {
			return nil, fmt.Errorf("check.yaml: objects[%d] needs select.kind", i)
		}
		var progs []cel.Program
		for j, a := range r.Assert {
			if strings.TrimSpace(a.Message) == "" {
				return nil, fmt.Errorf("check.yaml: objects[%d].assert[%d] needs a message", i, j)
			}
			ast, iss := env.Compile(a.Expr)
			if iss.Err() != nil {
				return nil, fmt.Errorf("check.yaml: objects[%d].assert[%d]: %w", i, j, iss.Err())
			}
			if ast.OutputType() != cel.BoolType && ast.OutputType() != cel.DynType {
				return nil, fmt.Errorf("check.yaml: objects[%d].assert[%d] must be a boolean expression", i, j)
			}
			prg, err := env.Program(ast)
			if err != nil {
				return nil, err
			}
			progs = append(progs, prg)
		}
		c.progs = append(c.progs, progs)
	}
	if err := c.compileSteps(); err != nil {
		return nil, err
	}
	return c, nil
}

// Run evaluates the check and returns the failures, written for the learner.
// An expression that cannot be evaluated (a field it reads is missing, say)
// counts as a failed assertion.
func (c *Compiled) Run(objs []map[string]any) []string {
	all := make([]any, len(objs))
	for i, o := range objs {
		all[i] = o
	}
	var failures []string
	for i, r := range c.Objects {
		var matches []map[string]any
		for _, o := range objs {
			if kind, _ := o["kind"].(string); kind != r.Select.Kind {
				continue
			}
			if r.Select.Name != "" && nameOf(o) != r.Select.Name {
				continue
			}
			matches = append(matches, o)
		}
		what := r.Select.Kind
		if r.Select.Name != "" {
			what += " " + r.Select.Name
		}
		switch {
		case len(matches) == 0:
			failures = append(failures, fmt.Sprintf("Expected a %s in your manifest, found none.", what))
			continue
		case len(matches) > 1 && !r.Many:
			failures = append(failures, fmt.Sprintf("Expected exactly one %s, found %d.", what, len(matches)))
			continue
		}
		for _, m := range matches {
			for j, a := range r.Assert {
				out, _, err := c.progs[i][j].Eval(map[string]any{"object": m, "objects": all})
				if err != nil || out.Value() != true {
					failures = append(failures, a.Message)
				}
			}
		}
	}
	return failures
}

func nameOf(o map[string]any) string {
	md, _ := o["metadata"].(map[string]any)
	n, _ := md["name"].(string)
	return n
}
