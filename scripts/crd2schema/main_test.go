package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const crd = `
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata: {name: widgets.example.io}
spec:
  group: example.io
  names: {kind: Widget}
  versions:
    - name: v1
      served: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              required: [size]
              properties:
                size: {type: integer, minimum: 1}
                port: {x-kubernetes-int-or-string: true}
                name: {type: string, nullable: true}
                free: {type: object, x-kubernetes-preserve-unknown-fields: true}
                tags: {type: object, additionalProperties: {type: string}}
                parts:
                  type: array
                  items:
                    type: object
                    properties:
                      id: {type: string}
    - name: v0
      served: false
      schema: {openAPIV3Schema: {type: object}}
`

func compile(t *testing.T, gz []byte) *jsonschema.Schema {
	t.Helper()
	zr, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(zr)
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("k8s://w.json", doc); err != nil {
		t.Fatal(err)
	}
	s, err := c.Compile("k8s://w.json")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestConvertMakesAStrictSchemaPerServedVersion(t *testing.T) {
	files, err := Convert([]byte("---\n"+crd), []string{"Widget"}, []string{"v1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files["widget-example-v1.json.gz"] == nil {
		t.Fatalf("files = %v (only the served version, named like the core schemas)", keys(files))
	}
	s := compile(t, files["widget-example-v1.json.gz"])
	obj := func(spec map[string]any) map[string]any {
		return map[string]any{"apiVersion": "example.io/v1", "kind": "Widget", "metadata": map[string]any{"name": "w", "labels": map[string]any{"a": "b"}}, "spec": spec}
	}
	good := obj(map[string]any{"size": 2, "port": "http", "name": nil, "free": map[string]any{"anything": 1}, "tags": map[string]any{"a": "b"}, "parts": []any{map[string]any{"id": "x"}}})
	if err := s.Validate(good); err != nil {
		t.Errorf("good object: %v", err)
	}
	for name, spec := range map[string]map[string]any{
		"a misspelled field":          {"size": 2, "sizee": 3},
		"a misspelled nested field":   {"size": 2, "parts": []any{map[string]any{"idd": "x"}}},
		"a missing required field":    {},
		"below the minimum":           {"size": 0},
		"the wrong type":              {"size": "two"},
		"a number that is not a port": {"size": 1, "port": 1.5},
	} {
		if err := s.Validate(obj(spec)); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if err := s.Validate(obj(map[string]any{"size": 1, "free": map[string]any{"x": map[string]any{"y": 1}}})); err != nil {
		t.Errorf("a field that preserves unknown fields stays open: %v", err)
	}
}

func TestConvertIsDeterministic(t *testing.T) {
	a, _ := Convert([]byte(crd), []string{"Widget"}, []string{"v1"})
	b, _ := Convert([]byte(crd), []string{"Widget"}, []string{"v1"})
	if !bytes.Equal(a["widget-example-v1.json.gz"], b["widget-example-v1.json.gz"]) {
		t.Fatal("the same input must give the same bytes: regenerating must not make a diff")
	}
}

func TestConvertRefusesAKindThatIsNotThere(t *testing.T) {
	if _, err := Convert([]byte(crd), []string{"Gadget"}, []string{"v1"}); err == nil || !strings.Contains(err.Error(), "Gadget") {
		t.Fatalf("err = %v", err)
	}
}

func keys(m map[string][]byte) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
