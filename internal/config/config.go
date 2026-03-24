package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Header represents a single HTTP header to inject into requests.
type Header struct {
	Key      string `yaml:"key"`
	Value    string `yaml:"value"`
	Disabled bool   `yaml:"disabled"`
}

// AuthAttribute is a key/value pair for a Postman auth method.
type AuthAttribute struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
	Type  string `yaml:"type,omitempty"`
}

// Auth holds the authentication configuration for a collection or folder.
type Auth struct {
	Type       string          `yaml:"type"`
	Attributes []AuthAttribute `yaml:"attributes,omitempty"`
	// Propagation controls how auth is applied to child folders and requests.
	// When set to "inherit", all folders and requests (except those with an
	// explicit folder_override) will have their auth cleared so they inherit
	// from the collection-level auth. Items with type "noauth" are left
	// untouched. Omit this field (or leave empty) to preserve the current
	// behaviour where each item keeps whatever auth was set previously.
	Propagation string `yaml:"propagation,omitempty"`
}

// Scripts holds pre-request and test scripts.
type Scripts struct {
	PreRequest string `yaml:"prerequest,omitempty"`
	Test       string `yaml:"test,omitempty"`
}

// FolderOverride lets you customise auth and scripts for a specific folder (tag).
type FolderOverride struct {
	Auth    *Auth    `yaml:"auth,omitempty"`
	Scripts *Scripts `yaml:"scripts,omitempty"`
}

// DocLinks configures automatic documentation link injection.
type DocLinks struct {
	BaseURL string `yaml:"base_url"`
}

// OpenAPIConfig holds all configuration for the openapi-sync command.
type OpenAPIConfig struct {
	Path            string                    `yaml:"path"`
	BaseURL         string                    `yaml:"base_url"`
	SanitizeEnums   bool                      `yaml:"sanitize_enums"`
	DocLinks        *DocLinks                 `yaml:"doc_links,omitempty"`
	CommonHeaders   []Header                  `yaml:"common_headers,omitempty"`
	Auth            *Auth                     `yaml:"auth,omitempty"`
	Scripts         *Scripts                  `yaml:"scripts,omitempty"`
	FolderOverrides map[string]FolderOverride `yaml:"folder_overrides,omitempty"`
}

// ExportConfig holds configuration for the export command.
type ExportConfig struct {
	OutputDir    string   `yaml:"output_dir"`
	Collections  []string `yaml:"collections,omitempty"`
	Environments []string `yaml:"environments,omitempty"`
	Pretty       bool     `yaml:"pretty"`
}

// ImportEntityConfig holds per-entity-type import configuration.
type ImportEntityConfig struct {
	Names    []string `yaml:"names,omitempty"`
	Strategy string   `yaml:"strategy,omitempty"`
}

// ImportConfig holds configuration for the import command.
type ImportConfig struct {
	InputDir     string              `yaml:"input_dir"`
	Collections  *ImportEntityConfig `yaml:"collections,omitempty"`
	Environments *ImportEntityConfig `yaml:"environments,omitempty"`
}

// Config is the top-level configuration structure.
type Config struct {
	PostmanAPIKey string `yaml:"postman_api_key"`
	WorkspaceID   string `yaml:"workspace_id"`

	OpenAPI *OpenAPIConfig `yaml:"openapi,omitempty"`
	Export  *ExportConfig  `yaml:"export,omitempty"`
	Import  *ImportConfig  `yaml:"import,omitempty"`
}

func defaultOpenAPIConfig() *OpenAPIConfig {
	return &OpenAPIConfig{
		Path:          "./openapi.json",
		BaseURL:       "{{baseUrl}}",
		SanitizeEnums: true,
	}
}

func defaultExportConfig() *ExportConfig {
	return &ExportConfig{
		OutputDir: "./postman-export",
		Pretty:    true,
	}
}

func defaultImportConfig() *ImportConfig {
	return &ImportConfig{
		InputDir: "./postman-export",
	}
}

// Load reads the YAML config file at path and applies environment variable
// overrides for secrets.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	cfg := &Config{
		OpenAPI: defaultOpenAPIConfig(),
		Export:  defaultExportConfig(),
		Import:  defaultImportConfig(),
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	// Restore nil sub-sections to defaults (e.g. if explicitly null in YAML).
	if cfg.OpenAPI == nil {
		cfg.OpenAPI = defaultOpenAPIConfig()
	}
	if cfg.Export == nil {
		cfg.Export = defaultExportConfig()
	}
	if cfg.Import == nil {
		cfg.Import = defaultImportConfig()
	}

	// Apply string-field defaults for fields that may have been zeroed.
	if cfg.OpenAPI.Path == "" {
		cfg.OpenAPI.Path = "./openapi.json"
	}
	if cfg.OpenAPI.BaseURL == "" {
		cfg.OpenAPI.BaseURL = "{{baseUrl}}"
	}
	if cfg.Export.OutputDir == "" {
		cfg.Export.OutputDir = "./postman-export"
	}
	if cfg.Import.InputDir == "" {
		cfg.Import.InputDir = "./postman-export"
	}

	// Environment variable overrides (always win over file values).
	if v := os.Getenv("POSTMAN_API_KEY"); v != "" {
		cfg.PostmanAPIKey = v
	}
	if v := os.Getenv("POSTMAN_WORKSPACE_ID"); v != "" {
		cfg.WorkspaceID = v
	}

	return cfg, nil
}

// ValidateGlobal checks that the required global fields are present.
func (c *Config) ValidateGlobal() error {
	if c.PostmanAPIKey == "" {
		return fmt.Errorf("postman_api_key is required (set in config or POSTMAN_API_KEY env var)")
	}
	if c.WorkspaceID == "" {
		return fmt.Errorf("workspace_id is required (set in config or POSTMAN_WORKSPACE_ID env var)")
	}
	return nil
}

// ValidateOpenAPISync checks the openapi section for the openapi-sync command.
func (c *Config) ValidateOpenAPISync() error {
	o := c.OpenAPI
	if o.Path == "" {
		return fmt.Errorf("openapi.path must not be empty")
	}
	if o.Auth != nil && o.Auth.Type == "" {
		return fmt.Errorf("openapi.auth.type must not be empty when auth is specified")
	}
	if o.Auth != nil && o.Auth.Propagation != "" && o.Auth.Propagation != "inherit" {
		return fmt.Errorf("openapi.auth.propagation must be %q or omitted, got %q", "inherit", o.Auth.Propagation)
	}
	return nil
}

// ValidateExport checks the export section for the export command.
func (c *Config) ValidateExport() error {
	e := c.Export
	if e.OutputDir == "" {
		return fmt.Errorf("export.output_dir must not be empty")
	}
	if len(e.Collections) == 0 && len(e.Environments) == 0 {
		return fmt.Errorf("export must specify at least one entity type (collections or environments)")
	}
	return nil
}

// ValidateImport checks the import section for the import command.
func (c *Config) ValidateImport() error {
	i := c.Import
	if i.InputDir == "" {
		return fmt.Errorf("import.input_dir must not be empty")
	}
	if i.Collections == nil && i.Environments == nil {
		return fmt.Errorf("import must specify at least one entity type (collections or environments)")
	}
	if i.Collections != nil {
		s := i.Collections.Strategy
		if s != "" && s != "overwrite" && s != "merge" {
			return fmt.Errorf("import.collections.strategy must be %q or %q, got %q", "overwrite", "merge", s)
		}
	}
	if i.Environments != nil {
		s := i.Environments.Strategy
		if s != "" && s != "overwrite" {
			return fmt.Errorf("import.environments.strategy must be %q or omitted, got %q", "overwrite", s)
		}
	}
	return nil
}
