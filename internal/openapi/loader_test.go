package openapi_test

import (
	"os"
	"path/filepath"
	"strings"
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

// writeFileInDir writes content to a named file (possibly in a subdirectory) inside dir.
func writeFileInDir(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("creating directory for %s: %v", name, err)
	}
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

// ── LoadAndBundle ─────────────────────────────────────────────────────────────

const minimalSpec = `openapi: "3.0.0"
info:
  title: TestAPI
  version: "1.0.0"
paths: {}
`

func TestLoadAndBundle_SingleFileYAML(t *testing.T) {
	path := writeFile(t, "openapi.yaml", minimalSpec)
	spec, err := openapi.LoadAndBundle(path, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec["openapi"] != "3.0.0" {
		t.Errorf("openapi = %v", spec["openapi"])
	}
}

func TestLoadAndBundle_SingleFileJSON(t *testing.T) {
	path := writeFile(t, "openapi.json", `{"openapi":"3.0.0","info":{"title":"T","version":"1.0"},"paths":{}}`)
	spec, err := openapi.LoadAndBundle(path, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec["openapi"] != "3.0.0" {
		t.Errorf("openapi = %v", spec["openapi"])
	}
}

func TestLoadAndBundle_ExternalRef(t *testing.T) {
	dir := t.TempDir()
	writeFileInDir(t, dir, "openapi.yaml", `openapi: "3.0.0"
info:
  title: TestAPI
  version: "1.0.0"
paths: {}
components:
  schemas:
    User:
      $ref: "./schemas/User.yaml"
`)
	writeFileInDir(t, dir, "schemas/User.yaml", `type: object
properties:
  id:
    type: string
`)

	spec, err := openapi.LoadAndBundle(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	components, ok := spec["components"].(map[string]any)
	if !ok {
		t.Fatalf("components is not a map: %T", spec["components"])
	}
	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		t.Fatalf("schemas is not a map: %T", components["schemas"])
	}
	user, ok := schemas["User"].(map[string]any)
	if !ok {
		t.Fatalf("User schema is not a map: %T", schemas["User"])
	}
	if user["type"] != "object" {
		t.Errorf("User.type = %v, want object — external $ref was not resolved", user["type"])
	}
}

func TestLoadAndBundle_DirectoryAutoDetect(t *testing.T) {
	dir := t.TempDir()
	writeFileInDir(t, dir, "openapi.yaml", minimalSpec)

	spec, err := openapi.LoadAndBundle(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec["openapi"] != "3.0.0" {
		t.Errorf("openapi = %v", spec["openapi"])
	}
}

func TestLoadAndBundle_DirectoryWithRootFile(t *testing.T) {
	dir := t.TempDir()
	writeFileInDir(t, dir, "main.yaml", minimalSpec)

	spec, err := openapi.LoadAndBundle(dir, "main.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec["openapi"] != "3.0.0" {
		t.Errorf("openapi = %v", spec["openapi"])
	}
}

func TestLoadAndBundle_DirectoryWithRootFileNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := openapi.LoadAndBundle(dir, "missing.yaml")
	if err == nil {
		t.Fatal("expected error for missing root file, got nil")
	}
}

func TestLoadAndBundle_DirectoryNoSpec(t *testing.T) {
	dir := t.TempDir()
	_, err := openapi.LoadAndBundle(dir, "")
	if err == nil {
		t.Fatal("expected error for directory with no spec, got nil")
	}
}

func TestLoadAndBundle_DirectoryAmbiguous(t *testing.T) {
	dir := t.TempDir()
	writeFileInDir(t, dir, "service-a.yaml", minimalSpec)
	writeFileInDir(t, dir, "service-b.yaml", minimalSpec)

	_, err := openapi.LoadAndBundle(dir, "")
	if err == nil {
		t.Fatal("expected error for ambiguous directory, got nil")
	}
	if !strings.Contains(err.Error(), "--openapi-root-file") {
		t.Errorf("error should suggest --openapi-root-file, got: %v", err)
	}
}

func TestLoadAndBundle_PathNotFound(t *testing.T) {
	_, err := openapi.LoadAndBundle(filepath.Join(t.TempDir(), "missing.yaml"), "")
	if err == nil {
		t.Fatal("expected error for missing path, got nil")
	}
}

func TestFindRootSpec_Priority(t *testing.T) {
	// When both openapi.yaml and swagger.json are present, openapi.yaml should win.
	dir := t.TempDir()
	writeFileInDir(t, dir, "swagger.json", `{"openapi":"3.0.0","info":{"title":"Swagger","version":"1.0"},"paths":{}}`)
	writeFileInDir(t, dir, "openapi.yaml", minimalSpec)

	spec, err := openapi.LoadAndBundle(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	info, ok := spec["info"].(map[string]any)
	if !ok {
		t.Fatalf("info is not a map")
	}
	if info["title"] != "TestAPI" {
		t.Errorf("title = %v, want TestAPI (openapi.yaml should take priority over swagger.json)", info["title"])
	}
}
