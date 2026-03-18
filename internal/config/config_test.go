package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/akyrey/postman-sync/internal/config"
)

// writeTemp writes content to a temp file and returns its path.
func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "postman-sync-*.yaml")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

func TestLoad_MinimalValid(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "key123"
workspace_id: "ws456"
openapi_path: "./api.json"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.PostmanAPIKey != "key123" {
		t.Errorf("PostmanAPIKey = %q, want %q", cfg.PostmanAPIKey, "key123")
	}
	if cfg.WorkspaceID != "ws456" {
		t.Errorf("WorkspaceID = %q, want %q", cfg.WorkspaceID, "ws456")
	}
	if cfg.OpenAPIPath != "./api.json" {
		t.Errorf("OpenAPIPath = %q, want %q", cfg.OpenAPIPath, "./api.json")
	}
}

func TestLoad_Defaults(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "key"
workspace_id: "ws"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.SanitizeEnums {
		t.Error("SanitizeEnums should default to true")
	}
	if cfg.BaseURL != "{{baseUrl}}" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "{{baseUrl}}")
	}
	if cfg.OpenAPIPath != "./openapi.json" {
		t.Errorf("OpenAPIPath = %q, want %q", cfg.OpenAPIPath, "./openapi.json")
	}
}

func TestLoad_EnvVarOverrides(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "file-key"
workspace_id: "file-ws"
`)
	t.Setenv("POSTMAN_API_KEY", "env-key")
	t.Setenv("POSTMAN_WORKSPACE_ID", "env-ws")

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.PostmanAPIKey != "env-key" {
		t.Errorf("PostmanAPIKey = %q, want env var value %q", cfg.PostmanAPIKey, "env-key")
	}
	if cfg.WorkspaceID != "env-ws" {
		t.Errorf("WorkspaceID = %q, want env var value %q", cfg.WorkspaceID, "env-ws")
	}
}

func TestLoad_MissingAPIKey(t *testing.T) {
	path := writeTemp(t, `workspace_id: "ws"`)
	// Make sure env var is not set.
	t.Setenv("POSTMAN_API_KEY", "")

	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for missing postman_api_key, got nil")
	}
}

func TestLoad_MissingWorkspaceID(t *testing.T) {
	path := writeTemp(t, `postman_api_key: "key"`)
	t.Setenv("POSTMAN_WORKSPACE_ID", "")

	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for missing workspace_id, got nil")
	}
}

func TestLoad_AuthTypeRequired(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "key"
workspace_id: "ws"
auth:
  attributes:
    - key: token
      value: "{{tok}}"
`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for auth without type, got nil")
	}
}

func TestLoad_FullConfig(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "key"
workspace_id: "ws"
openapi_path: "./spec.yaml"
base_url: "{{myBase}}"
sanitize_enums: false
doc_links:
  base_url: "https://docs.example.com/#tag/"
common_headers:
  - key: X-Tenant
    value: "{{tenantId}}"
    disabled: false
auth:
  type: bearer
  attributes:
    - key: token
      value: "{{token}}"
      type: string
scripts:
  prerequest: "console.log('pre');"
  test: "pm.test('ok', () => {});"
folder_overrides:
  Pets:
    auth:
      type: noauth
    scripts:
      test: "pm.test('pets', () => {});"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.OpenAPIPath != "./spec.yaml" {
		t.Errorf("OpenAPIPath = %q", cfg.OpenAPIPath)
	}
	if cfg.SanitizeEnums {
		t.Error("SanitizeEnums should be false")
	}
	if cfg.DocLinks == nil || cfg.DocLinks.BaseURL != "https://docs.example.com/#tag/" {
		t.Errorf("DocLinks.BaseURL mismatch: %+v", cfg.DocLinks)
	}
	if len(cfg.CommonHeaders) != 1 || cfg.CommonHeaders[0].Key != "X-Tenant" {
		t.Errorf("CommonHeaders = %+v", cfg.CommonHeaders)
	}
	if cfg.Auth == nil || cfg.Auth.Type != "bearer" {
		t.Errorf("Auth = %+v", cfg.Auth)
	}
	if len(cfg.Auth.Attributes) != 1 || cfg.Auth.Attributes[0].Key != "token" {
		t.Errorf("Auth.Attributes = %+v", cfg.Auth.Attributes)
	}
	if cfg.Scripts == nil || cfg.Scripts.PreRequest == "" || cfg.Scripts.Test == "" {
		t.Errorf("Scripts = %+v", cfg.Scripts)
	}
	if _, ok := cfg.FolderOverrides["Pets"]; !ok {
		t.Error("FolderOverrides missing 'Pets' entry")
	}
}

func TestLoad_AuthPropagationInherit(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "key"
workspace_id: "ws"
auth:
  type: bearer
  propagation: inherit
  attributes:
    - key: token
      value: "{{tok}}"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Auth == nil {
		t.Fatal("Auth should not be nil")
	}
	if cfg.Auth.Propagation != "inherit" {
		t.Errorf("Auth.Propagation = %q, want %q", cfg.Auth.Propagation, "inherit")
	}
}

func TestLoad_AuthPropagationInvalidValue(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "key"
workspace_id: "ws"
auth:
  type: bearer
  propagation: copy
`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for unsupported propagation value, got nil")
	}
}

func TestLoad_AuthPropagationOmitted(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "key"
workspace_id: "ws"
auth:
  type: bearer
  attributes:
    - key: token
      value: "{{tok}}"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Auth.Propagation != "" {
		t.Errorf("Auth.Propagation should be empty when omitted, got %q", cfg.Auth.Propagation)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := config.Load(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeTemp(t, `{bad yaml: [}`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}
