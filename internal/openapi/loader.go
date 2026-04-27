package openapi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pb33f/libopenapi/bundler"
	"github.com/pb33f/libopenapi/datamodel"
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

// LoadAndBundle reads an OpenAPI specification from path (a file or directory)
// and returns it as a generic map with all external $ref references resolved and
// inlined. When path is a directory, the root spec file is auto-detected using
// well-known names (openapi.yaml, openapi.json, swagger.yaml, …) or the
// rootFile hint. When path is a file, rootFile is ignored.
func LoadAndBundle(path, rootFile string) (map[string]any, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("accessing OpenAPI path %q: %w", path, err)
	}

	root := path
	if info.IsDir() {
		root, err = findRootSpec(path, rootFile)
		if err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(root)
	if err != nil {
		return nil, fmt.Errorf("reading OpenAPI file %q: %w", root, err)
	}

	cfg := &datamodel.DocumentConfiguration{
		BasePath:            filepath.Dir(root),
		SpecFilePath:        filepath.Base(root),
		AllowFileReferences: true,
	}

	bundled, err := bundler.BundleBytes(data, cfg)
	if err != nil {
		return nil, fmt.Errorf("bundling OpenAPI spec %q: %w", root, err)
	}

	var spec map[string]any
	if err := yaml.Unmarshal(bundled, &spec); err != nil {
		return nil, fmt.Errorf("parsing bundled OpenAPI spec: %w", err)
	}

	spec, err = normalise(spec)
	if err != nil {
		return nil, fmt.Errorf("normalising OpenAPI spec: %w", err)
	}

	return spec, nil
}

// findRootSpec locates the root OpenAPI spec file within dir. When rootFile is
// non-empty it is joined with dir and returned directly (error if not found).
// Otherwise well-known names are tried first; if none match, the single
// .json/.yaml/.yml file at the top level of the directory is used.
func findRootSpec(dir, rootFile string) (string, error) {
	if rootFile != "" {
		candidate := filepath.Join(dir, rootFile)
		if _, err := os.Stat(candidate); err != nil {
			return "", fmt.Errorf("root file %q not found in %q", rootFile, dir)
		}
		return candidate, nil
	}

	wellKnown := []string{
		"openapi.yaml", "openapi.yml", "openapi.json",
		"swagger.yaml", "swagger.yml", "swagger.json",
	}
	for _, name := range wellKnown {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("reading directory %q: %w", dir, err)
	}

	var candidates []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".json", ".yaml", ".yml":
			candidates = append(candidates, e.Name())
		}
	}

	switch len(candidates) {
	case 0:
		return "", fmt.Errorf("no OpenAPI spec files found in %q", dir)
	case 1:
		return filepath.Join(dir, candidates[0]), nil
	default:
		return "", fmt.Errorf(
			"multiple spec files found in %q (%s); specify the root file via --openapi-root-file or openapi.root_file config",
			dir, strings.Join(candidates, ", "),
		)
	}
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
