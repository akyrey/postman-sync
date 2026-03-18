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

// Config is the top-level configuration structure.
type Config struct {
	PostmanAPIKey string `yaml:"postman_api_key"`
	WorkspaceID   string `yaml:"workspace_id"`

	OpenAPIPath string `yaml:"openapi_path"`
	BaseURL     string `yaml:"base_url"`

	SanitizeEnums bool      `yaml:"sanitize_enums"`
	DocLinks      *DocLinks `yaml:"doc_links,omitempty"`

	CommonHeaders []Header `yaml:"common_headers,omitempty"`

	Auth    *Auth    `yaml:"auth,omitempty"`
	Scripts *Scripts `yaml:"scripts,omitempty"`

	FolderOverrides map[string]FolderOverride `yaml:"folder_overrides,omitempty"`
}

// Load reads the YAML config file at path and applies environment variable
// overrides for secrets.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	cfg := &Config{
		SanitizeEnums: true, // default on
		BaseURL:       "{{baseUrl}}",
		OpenAPIPath:   "./openapi.json",
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	// Environment variable overrides (always win over file values).
	if v := os.Getenv("POSTMAN_API_KEY"); v != "" {
		cfg.PostmanAPIKey = v
	}
	if v := os.Getenv("POSTMAN_WORKSPACE_ID"); v != "" {
		cfg.WorkspaceID = v
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.PostmanAPIKey == "" {
		return fmt.Errorf("postman_api_key is required (set in config or POSTMAN_API_KEY env var)")
	}
	if c.WorkspaceID == "" {
		return fmt.Errorf("workspace_id is required (set in config or POSTMAN_WORKSPACE_ID env var)")
	}
	if c.OpenAPIPath == "" {
		return fmt.Errorf("openapi_path must not be empty")
	}
	if c.Auth != nil && c.Auth.Type == "" {
		return fmt.Errorf("auth.type must not be empty when auth is specified")
	}
	if c.Auth != nil && c.Auth.Propagation != "" && c.Auth.Propagation != "inherit" {
		return fmt.Errorf("auth.propagation must be %q or omitted, got %q", "inherit", c.Auth.Propagation)
	}
	return nil
}
