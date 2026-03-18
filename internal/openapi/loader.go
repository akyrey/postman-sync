package openapi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads an OpenAPI specification from path (JSON or YAML) and returns
// it as a generic map suitable for marshalling back to JSON.
func Load(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading OpenAPI file %q: %w", path, err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	var spec map[string]any

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("parsing YAML OpenAPI file %q: %w", path, err)
		}
		// yaml.v3 may produce map[string]interface{} with nested map[interface{}]interface{}
		// for some edge cases. Normalise by round-tripping through JSON.
		spec, err = normalise(spec)
		if err != nil {
			return nil, fmt.Errorf("normalising OpenAPI spec: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("parsing JSON OpenAPI file %q: %w", path, err)
		}
	default:
		return nil, fmt.Errorf("unsupported OpenAPI file extension %q (use .json, .yaml, or .yml)", ext)
	}

	return spec, nil
}

// SanitizeEnums walks the spec recursively and replaces every "enum" array
// with ["<enum>"], reducing noise in Postman diffs.
func SanitizeEnums(spec any) {
	switch v := spec.(type) {
	case map[string]any:
		for k, val := range v {
			if k == "enum" {
				if _, ok := val.([]any); ok {
					v[k] = []any{"<enum>"}
					continue
				}
			}
			SanitizeEnums(val)
		}
	case []any:
		for _, item := range v {
			SanitizeEnums(item)
		}
	}
}

// normalise converts any YAML-decoded map to a JSON-compatible map by
// round-tripping through JSON encoding/decoding.
func normalise(v map[string]any) (map[string]any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}
