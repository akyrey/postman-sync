package postman

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultPostmanBaseURL = "https://api.getpostman.com"

// Client is a minimal Postman API client.
type Client struct {
	apiKey      string
	workspaceID string
	baseURL     string
	http        *http.Client
}

// NewClient creates a new Postman API client.
func NewClient(apiKey, workspaceID string) *Client {
	return &Client{
		apiKey:      apiKey,
		workspaceID: workspaceID,
		baseURL:     defaultPostmanBaseURL,
		http: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// withBaseURL returns a copy of the client using a custom base URL.
// Intended for testing only.
func (c *Client) withBaseURL(u string) *Client {
	copy := *c
	copy.baseURL = u
	return &copy
}

// ImportOpenAPI uploads an OpenAPI spec to Postman and returns the UID of
// the temporary collection that Postman generates.
func (c *Client) ImportOpenAPI(spec map[string]any) (string, error) {
	payload := map[string]any{
		"type":    "json",
		"input":   spec,
		"options": map[string]any{},
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshalling import payload: %w", err)
	}

	url := fmt.Sprintf("%s/import/openapi?workspace=%s", c.baseURL, c.workspaceID)
	resp, err := c.do("POST", url, b)
	if err != nil {
		return "", fmt.Errorf("importing OpenAPI spec: %w", err)
	}

	var result struct {
		Collections []struct {
			UID string `json:"uid"`
		} `json:"collections"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("parsing import response: %w", err)
	}
	if len(result.Collections) == 0 {
		return "", fmt.Errorf("import returned no collections")
	}
	return result.Collections[0].UID, nil
}

// GetCollection fetches a full collection by its ID or UID.
func (c *Client) GetCollection(id string) (*CollectionWrapper, error) {
	url := fmt.Sprintf("%s/collections/%s", c.baseURL, id)
	b, err := c.do("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching collection %q: %w", id, err)
	}

	var wrapper CollectionWrapper
	if err := json.Unmarshal(b, &wrapper); err != nil {
		return nil, fmt.Errorf("parsing collection %q: %w", id, err)
	}
	return &wrapper, nil
}

// DeleteCollection deletes a collection by ID/UID. Errors are logged but not fatal.
func (c *Client) DeleteCollection(id string) error {
	url := fmt.Sprintf("%s/collections/%s", c.baseURL, id)
	if _, err := c.do("DELETE", url, nil); err != nil {
		return fmt.Errorf("deleting collection %q: %w", id, err)
	}
	return nil
}

// UpdateCollection replaces an existing collection with new data.
func (c *Client) UpdateCollection(id string, data *CollectionWrapper) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshalling collection: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s", c.baseURL, id)
	if _, err := c.do("PUT", url, b); err != nil {
		return fmt.Errorf("updating collection %q: %w", id, err)
	}
	return nil
}

// CreateCollection creates a new collection in the workspace.
func (c *Client) CreateCollection(data *CollectionWrapper) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshalling collection: %w", err)
	}

	url := fmt.Sprintf("%s/collections?workspace=%s", c.baseURL, c.workspaceID)
	if _, err := c.do("POST", url, b); err != nil {
		return fmt.Errorf("creating collection: %w", err)
	}
	return nil
}

// GetWorkspaceCollections returns a map of collection name → collection ID
// for all collections in the configured workspace.
func (c *Client) GetWorkspaceCollections() (map[string]string, error) {
	url := fmt.Sprintf("%s/workspaces/%s", c.baseURL, c.workspaceID)
	b, err := c.do("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching workspace %q: %w", c.workspaceID, err)
	}

	var result struct {
		Workspace struct {
			Collections []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"collections"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("parsing workspace response: %w", err)
	}

	m := make(map[string]string, len(result.Workspace.Collections))
	for _, col := range result.Workspace.Collections {
		m[col.Name] = col.ID
	}
	return m, nil
}

// GetWorkspaceEnvironments returns a map of environment name → environment UID
// for all environments in the configured workspace.
func (c *Client) GetWorkspaceEnvironments() (map[string]string, error) {
	url := fmt.Sprintf("%s/workspaces/%s", c.baseURL, c.workspaceID)
	b, err := c.do("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching workspace %q: %w", c.workspaceID, err)
	}

	var result struct {
		Workspace struct {
			Environments []struct {
				ID   string `json:"id"`
				UID  string `json:"uid"`
				Name string `json:"name"`
			} `json:"environments"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("parsing workspace response: %w", err)
	}

	m := make(map[string]string, len(result.Workspace.Environments))
	for _, env := range result.Workspace.Environments {
		id := env.UID
		if id == "" {
			id = env.ID
		}
		m[env.Name] = id
	}
	return m, nil
}

// GetEnvironment fetches a full environment by its ID or UID.
func (c *Client) GetEnvironment(id string) (*EnvironmentWrapper, error) {
	url := fmt.Sprintf("%s/environments/%s", c.baseURL, id)
	b, err := c.do("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching environment %q: %w", id, err)
	}

	var wrapper EnvironmentWrapper
	if err := json.Unmarshal(b, &wrapper); err != nil {
		return nil, fmt.Errorf("parsing environment %q: %w", id, err)
	}
	return &wrapper, nil
}

// CreateEnvironment creates a new environment in the workspace.
func (c *Client) CreateEnvironment(env *EnvironmentWrapper) error {
	b, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshalling environment: %w", err)
	}

	url := fmt.Sprintf("%s/environments?workspace=%s", c.baseURL, c.workspaceID)
	if _, err := c.do("POST", url, b); err != nil {
		return fmt.Errorf("creating environment: %w", err)
	}
	return nil
}

// UpdateEnvironment replaces an existing environment with new data.
func (c *Client) UpdateEnvironment(id string, env *EnvironmentWrapper) error {
	b, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshalling environment: %w", err)
	}

	url := fmt.Sprintf("%s/environments/%s", c.baseURL, id)
	if _, err := c.do("PUT", url, b); err != nil {
		return fmt.Errorf("updating environment %q: %w", id, err)
	}
	return nil
}

// DeleteEnvironment deletes an environment by ID/UID.
func (c *Client) DeleteEnvironment(id string) error {
	url := fmt.Sprintf("%s/environments/%s", c.baseURL, id)
	if _, err := c.do("DELETE", url, nil); err != nil {
		return fmt.Errorf("deleting environment %q: %w", id, err)
	}
	return nil
}

// do executes an HTTP request with the Postman API key header and returns the
// response body. Non-2xx responses are returned as errors.
func (c *Client) do(method, url string, body []byte) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-API-Key", c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d %s: %s", resp.StatusCode, resp.Status, truncate(string(respBody), 300))
	}

	return respBody, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
