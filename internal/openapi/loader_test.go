package openapi_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/akyrey/postman-sync/internal/openapi"
)

// writeFile writes content to a named file inside a temp dir and returns the path.
func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
	return path
}

// ── Load ──────────────────────────────────────────────────────────────────────

func TestLoad_JSON(t *testing.T) {
	path := writeFile(t, "spec.json", `{"openapi":"3.0.0","info":{"title":"Test"}}`)
	spec, err := openapi.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec["openapi"] != "3.0.0" {
		t.Errorf("openapi field = %v", spec["openapi"])
	}
	info, ok := spec["info"].(map[string]any)
	if !ok {
		t.Fatalf("info is not a map: %T", spec["info"])
	}
	if info["title"] != "Test" {
		t.Errorf("title = %v", info["title"])
	}
}

func TestLoad_YAML(t *testing.T) {
	path := writeFile(t, "spec.yaml", `
openapi: "3.0.0"
info:
  title: MyAPI
paths: {}
`)
	spec, err := openapi.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec["openapi"] != "3.0.0" {
		t.Errorf("openapi = %v", spec["openapi"])
	}
	info := spec["info"].(map[string]any)
	if info["title"] != "MyAPI" {
		t.Errorf("title = %v", info["title"])
	}
}

func TestLoad_YML_extension(t *testing.T) {
	path := writeFile(t, "spec.yml", `openapi: "3.1.0"
info:
  title: YML
`)
	spec, err := openapi.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec["openapi"] != "3.1.0" {
		t.Errorf("openapi = %v", spec["openapi"])
	}
}

func TestLoad_UnsupportedExtension(t *testing.T) {
	path := writeFile(t, "spec.toml", `openapi = "3.0.0"`)
	_, err := openapi.Load(path)
	if err == nil {
		t.Fatal("expected error for unsupported extension, got nil")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := openapi.Load(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	path := writeFile(t, "spec.json", `{invalid json`)
	_, err := openapi.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeFile(t, "spec.yaml", `{bad yaml: [}`)
	_, err := openapi.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

// ── SanitizeEnums ────────────────────────────────────────────────────────────

func TestSanitizeEnums_ReplacesTopLevel(t *testing.T) {
	spec := map[string]any{
		"enum": []any{"active", "inactive", "pending"},
	}
	openapi.SanitizeEnums(spec)
	got := spec["enum"].([]any)
	if len(got) != 1 || got[0] != "<enum>" {
		t.Errorf("enum = %v, want [<enum>]", got)
	}
}

func TestSanitizeEnums_ReplacesNested(t *testing.T) {
	spec := map[string]any{
		"components": map[string]any{
			"schemas": map[string]any{
				"Status": map[string]any{
					"type": "string",
					"enum": []any{"on", "off"},
				},
			},
		},
	}
	openapi.SanitizeEnums(spec)

	schemas := spec["components"].(map[string]any)["schemas"].(map[string]any)
	status := schemas["Status"].(map[string]any)
	got := status["enum"].([]any)
	if len(got) != 1 || got[0] != "<enum>" {
		t.Errorf("nested enum = %v, want [<enum>]", got)
	}
}

func TestSanitizeEnums_IgnoresNonArrayEnum(t *testing.T) {
	spec := map[string]any{
		"enum": "not-an-array",
	}
	openapi.SanitizeEnums(spec)
	// Should not be replaced because it's not a []any.
	if spec["enum"] != "not-an-array" {
		t.Errorf("enum should be unchanged, got %v", spec["enum"])
	}
}

func TestSanitizeEnums_HandlesArrayValues(t *testing.T) {
	spec := map[string]any{
		"items": []any{
			map[string]any{"enum": []any{"a", "b"}},
			map[string]any{"type": "string"},
		},
	}
	openapi.SanitizeEnums(spec)
	first := spec["items"].([]any)[0].(map[string]any)
	got := first["enum"].([]any)
	if len(got) != 1 || got[0] != "<enum>" {
		t.Errorf("enum in array = %v, want [<enum>]", got)
	}
}

func TestSanitizeEnums_EmptySpec(t *testing.T) {
	spec := map[string]any{}
	openapi.SanitizeEnums(spec) // should not panic
}

func TestSanitizeEnums_PreservesOtherKeys(t *testing.T) {
	spec := map[string]any{
		"type":        "string",
		"description": "a status",
		"enum":        []any{"x", "y"},
	}
	openapi.SanitizeEnums(spec)
	if spec["type"] != "string" {
		t.Errorf("type changed unexpectedly: %v", spec["type"])
	}
	if spec["description"] != "a status" {
		t.Errorf("description changed unexpectedly: %v", spec["description"])
	}
}
