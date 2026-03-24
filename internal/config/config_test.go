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
	_ = f.Close()
	return f.Name()
}

func TestLoad_GlobalOnly(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "key123"
workspace_id: "ws456"
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
}

func TestLoad_OpenAPIDefaults(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "key"
workspace_id: "ws"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.OpenAPI == nil {
		t.Fatal("OpenAPI should not be nil")
		return
	}
	if !cfg.OpenAPI.SanitizeEnums {
		t.Error("OpenAPI.SanitizeEnums should default to true")
	}
	if cfg.OpenAPI.BaseURL != "{{baseUrl}}" {
		t.Errorf("OpenAPI.BaseURL = %q, want %q", cfg.OpenAPI.BaseURL, "{{baseUrl}}")
	}
	if cfg.OpenAPI.Path != "./openapi.json" {
		t.Errorf("OpenAPI.Path = %q, want %q", cfg.OpenAPI.Path, "./openapi.json")
	}
}

func TestLoad_ExportDefaults(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "key"
workspace_id: "ws"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Export == nil {
		t.Fatal("Export should not be nil")
		return
	}
	if cfg.Export.OutputDir != "./postman-export" {
		t.Errorf("Export.OutputDir = %q, want %q", cfg.Export.OutputDir, "./postman-export")
	}
	if !cfg.Export.Pretty {
		t.Error("Export.Pretty should default to true")
	}
}

func TestLoad_ImportDefaults(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "key"
workspace_id: "ws"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Import == nil {
		t.Fatal("Import should not be nil")
		return
	}
	if cfg.Import.InputDir != "./postman-export" {
		t.Errorf("Import.InputDir = %q, want %q", cfg.Import.InputDir, "./postman-export")
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

func TestLoad_OpenAPISection(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "key"
workspace_id: "ws"
openapi:
  path: "./spec.yaml"
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
	o := cfg.OpenAPI
	if o == nil {
		t.Fatal("OpenAPI should not be nil")
		return
	}
	if o.Path != "./spec.yaml" {
		t.Errorf("OpenAPI.Path = %q", o.Path)
	}
	if o.SanitizeEnums {
		t.Error("OpenAPI.SanitizeEnums should be false")
	}
	if o.DocLinks == nil || o.DocLinks.BaseURL != "https://docs.example.com/#tag/" {
		t.Errorf("OpenAPI.DocLinks.BaseURL mismatch: %+v", o.DocLinks)
	}
	if len(o.CommonHeaders) != 1 || o.CommonHeaders[0].Key != "X-Tenant" {
		t.Errorf("OpenAPI.CommonHeaders = %+v", o.CommonHeaders)
	}
	if o.Auth == nil || o.Auth.Type != "bearer" {
		t.Errorf("OpenAPI.Auth = %+v", o.Auth)
	}
	if len(o.Auth.Attributes) != 1 || o.Auth.Attributes[0].Key != "token" {
		t.Errorf("OpenAPI.Auth.Attributes = %+v", o.Auth.Attributes)
	}
	if o.Scripts == nil || o.Scripts.PreRequest == "" || o.Scripts.Test == "" {
		t.Errorf("OpenAPI.Scripts = %+v", o.Scripts)
	}
	if _, ok := o.FolderOverrides["Pets"]; !ok {
		t.Error("OpenAPI.FolderOverrides missing 'Pets' entry")
	}
}

func TestLoad_ExportSection(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "key"
workspace_id: "ws"
export:
  output_dir: "./my-export"
  collections:
    - "My API"
    - "Other"
  environments:
    - "all"
  pretty: false
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	e := cfg.Export
	if e.OutputDir != "./my-export" {
		t.Errorf("Export.OutputDir = %q", e.OutputDir)
	}
	if len(e.Collections) != 2 {
		t.Errorf("Export.Collections = %v", e.Collections)
	}
	if len(e.Environments) != 1 || e.Environments[0] != "all" {
		t.Errorf("Export.Environments = %v", e.Environments)
	}
	if e.Pretty {
		t.Error("Export.Pretty should be false")
	}
}

func TestLoad_ImportSection(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "key"
workspace_id: "ws"
import:
  input_dir: "./my-import"
  collections:
    names:
      - "all"
    strategy: "merge"
  environments:
    names:
      - "Production"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	i := cfg.Import
	if i.InputDir != "./my-import" {
		t.Errorf("Import.InputDir = %q", i.InputDir)
	}
	if i.Collections == nil || i.Collections.Strategy != "merge" {
		t.Errorf("Import.Collections = %+v", i.Collections)
	}
	if i.Environments == nil || len(i.Environments.Names) != 1 {
		t.Errorf("Import.Environments = %+v", i.Environments)
	}
}

func TestValidateGlobal_MissingAPIKey(t *testing.T) {
	path := writeTemp(t, `workspace_id: "ws"`)
	t.Setenv("POSTMAN_API_KEY", "")

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if err := cfg.ValidateGlobal(); err == nil {
		t.Fatal("expected error for missing postman_api_key, got nil")
	}
}

func TestValidateGlobal_MissingWorkspaceID(t *testing.T) {
	path := writeTemp(t, `postman_api_key: "key"`)
	t.Setenv("POSTMAN_WORKSPACE_ID", "")

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if err := cfg.ValidateGlobal(); err == nil {
		t.Fatal("expected error for missing workspace_id, got nil")
	}
}

func TestValidateOpenAPISync_AuthTypeRequired(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "key"
workspace_id: "ws"
openapi:
  auth:
    attributes:
      - key: token
        value: "{{tok}}"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if err := cfg.ValidateOpenAPISync(); err == nil {
		t.Fatal("expected error for auth without type, got nil")
	}
}

func TestValidateOpenAPISync_PropagationInherit(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "key"
workspace_id: "ws"
openapi:
  auth:
    type: bearer
    propagation: inherit
    attributes:
      - key: token
        value: "{{tok}}"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if err := cfg.ValidateOpenAPISync(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.OpenAPI.Auth.Propagation != "inherit" {
		t.Errorf("Auth.Propagation = %q, want %q", cfg.OpenAPI.Auth.Propagation, "inherit")
	}
}

func TestValidateOpenAPISync_PropagationInvalidValue(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "key"
workspace_id: "ws"
openapi:
  auth:
    type: bearer
    propagation: copy
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if err := cfg.ValidateOpenAPISync(); err == nil {
		t.Fatal("expected error for unsupported propagation value, got nil")
	}
}

func TestValidateExport_NoEntities(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "key"
workspace_id: "ws"
export:
  output_dir: "./out"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if err := cfg.ValidateExport(); err == nil {
		t.Fatal("expected error when no entity types selected, got nil")
	}
}

func TestValidateImport_InvalidStrategy(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "key"
workspace_id: "ws"
import:
  collections:
    names: ["all"]
    strategy: "clone"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if err := cfg.ValidateImport(); err == nil {
		t.Fatal("expected error for invalid import strategy, got nil")
	}
}

func TestValidateImport_EnvironmentsNoMerge(t *testing.T) {
	path := writeTemp(t, `
postman_api_key: "key"
workspace_id: "ws"
import:
  environments:
    names: ["all"]
    strategy: "merge"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if err := cfg.ValidateImport(); err == nil {
		t.Fatal("expected error for merge strategy on environments, got nil")
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
