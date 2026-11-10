// Command crd2schema turns the CustomResourceDefinitions of a manifest into the
// strict JSON Schema files that manifest-mode exercises are validated against,
// one gzipped file per kind and version, named like the core ones
// (httproute-gateway-v1.json.gz). Nothing is downloaded: the input is a pinned
// manifest of the repository (addons/gateway-api/standard-install.yaml).
//
//	go run ./scripts/crd2schema -in addons/gateway-api/standard-install.yaml -out schemas -kinds HTTPRoute,Gateway
//
// "Strict" is what the core schemas are: an object with properties lists them
// all, and a field it does not list is an error (that is what catches a
// misspelled sectionNmae). A field marked x-kubernetes-preserve-unknown-fields
// stays open.
package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/cleonte/k8slings/k8slings/schema"
	"sigs.k8s.io/yaml"
)

func main() {
	in := flag.String("in", "", "a manifest with CustomResourceDefinitions")
	out := flag.String("out", "schemas", "the folder to write the .json.gz files to")
	kinds := flag.String("kinds", "", "the kinds to convert, separated by commas")
	versions := flag.String("versions", "v1", "the versions to convert, separated by commas")
	flag.Parse()
	if *in == "" || *kinds == "" {
		fmt.Fprintln(os.Stderr, "usage: crd2schema -in manifest.yaml -out schemas -kinds HTTPRoute,Gateway")
		os.Exit(2)
	}
	data, err := os.ReadFile(*in)
	if err != nil {
		fatal(err)
	}
	files, err := Convert(data, strings.Split(*kinds, ","), strings.Split(*versions, ","))
	if err != nil {
		fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(*out, name), content, 0o644); err != nil {
			fatal(err)
		}
	}
	fmt.Printf("wrote %d schemas from %s\n", len(files), *in)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "crd2schema:", err)
	os.Exit(1)
}

var docSeparator = regexp.MustCompile(`(?m)^---[ \t]*$`)

// Convert returns the gzipped schema file of the wanted versions of the wanted
// kinds, by file name.
func Convert(manifest []byte, kinds, versions []string) (map[string][]byte, error) {
	out := map[string][]byte{}
	found := map[string]bool{}
	for _, doc := range docSeparator.Split(string(manifest), -1) {
		var crd struct {
			Kind string `json:"kind"`
			Spec struct {
				Group string `json:"group"`
				Names struct {
					Kind string `json:"kind"`
				} `json:"names"`
				Versions []struct {
					Name   string `json:"name"`
					Served bool   `json:"served"`
					Schema struct {
						OpenAPIV3Schema map[string]any `json:"openAPIV3Schema"`
					} `json:"schema"`
				} `json:"versions"`
			} `json:"spec"`
		}
		if err := yaml.Unmarshal([]byte(doc), &crd); err != nil {
			return nil, err
		}
		if crd.Kind != "CustomResourceDefinition" || !slices.Contains(kinds, crd.Spec.Names.Kind) {
			continue
		}
		for _, v := range crd.Spec.Versions {
			if !v.Served || v.Schema.OpenAPIV3Schema == nil || !slices.Contains(versions, v.Name) {
				continue
			}
			s := toSchema(v.Schema.OpenAPIV3Schema).(map[string]any)
			props, _ := s["properties"].(map[string]any)
			if props == nil {
				props = map[string]any{}
				s["properties"] = props
			}
			props["apiVersion"] = map[string]any{"type": "string"}
			props["kind"] = map[string]any{"type": "string"}
			if _, ok := props["metadata"]; !ok {
				props["metadata"] = map[string]any{"type": "object"}
			}
			raw, err := json.Marshal(s) // sorted keys: the same bytes every time
			if err != nil {
				return nil, err
			}
			var buf bytes.Buffer
			zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
			if _, err := zw.Write(raw); err != nil {
				return nil, err
			}
			if err := zw.Close(); err != nil {
				return nil, err
			}
			out[schema.FileName(crd.Spec.Group+"/"+v.Name, crd.Spec.Names.Kind)+".json.gz"] = buf.Bytes()
			found[crd.Spec.Names.Kind] = true
		}
	}
	for _, k := range kinds {
		if !found[k] {
			return nil, fmt.Errorf("no CustomResourceDefinition for %s in the manifest", k)
		}
	}
	return out, nil
}

// keep are the validation keywords that carry over unchanged.
var keep = []string{"type", "format", "enum", "pattern", "minLength", "maxLength", "minimum", "maximum",
	"exclusiveMinimum", "exclusiveMaximum", "multipleOf", "minItems", "maxItems", "uniqueItems",
	"minProperties", "maxProperties", "required"}

func toSchema(in any) any {
	m, ok := in.(map[string]any)
	if !ok {
		return in
	}
	out := map[string]any{}
	for _, k := range keep {
		if v, ok := m[k]; ok {
			out[k] = v
		}
	}
	if props, ok := m["properties"].(map[string]any); ok {
		conv := map[string]any{}
		for name, p := range props {
			conv[name] = toSchema(p)
		}
		out["properties"] = conv
	}
	if items, ok := m["items"]; ok {
		out["items"] = toSchema(items)
	}
	switch ap := m["additionalProperties"].(type) {
	case map[string]any:
		out["additionalProperties"] = toSchema(ap)
	case bool:
		out["additionalProperties"] = ap
	}
	for _, k := range []string{"allOf", "anyOf", "oneOf"} {
		if list, ok := m[k].([]any); ok {
			conv := make([]any, len(list))
			for i, e := range list {
				conv[i] = toSchema(e)
			}
			out[k] = conv
		}
	}
	if not, ok := m["not"]; ok {
		out["not"] = toSchema(not)
	}
	if b, _ := m["x-kubernetes-int-or-string"].(bool); b {
		delete(out, "type")
		out["anyOf"] = []any{map[string]any{"type": "integer"}, map[string]any{"type": "string"}}
	}
	if b, _ := m["nullable"].(bool); b {
		if t, ok := out["type"].(string); ok {
			out["type"] = []any{t, "null"}
		}
	}
	preserve, _ := m["x-kubernetes-preserve-unknown-fields"].(bool)
	if _, has := out["additionalProperties"]; !has && !preserve && out["type"] == "object" {
		if _, hasProps := out["properties"]; hasProps {
			out["additionalProperties"] = false
		}
	}
	return out
}
