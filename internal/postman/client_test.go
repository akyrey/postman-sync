package postman

// White-box tests: same package so we can access withBaseURL.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestClient creates a Client pointed at the given httptest server.
func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	c := NewClient("test-api-key", "test-workspace")
	return c.withBaseURL(server.URL)
}

// ── ImportOpenAPI ─────────────────────────────────────────────────────────────

func TestImportOpenAPI_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("X-API-Key") != "test-api-key" {
			t.Errorf("X-API-Key header missing or wrong: %s", r.Header.Get("X-API-Key"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"collections": []map[string]any{{"uid": "abc-123"}},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	uid, err := c.ImportOpenAPI(map[string]any{"openapi": "3.0.0"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uid != "abc-123" {
		t.Errorf("uid = %q, want %q", uid, "abc-123")
	}
}

func TestImportOpenAPI_EmptyCollections(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"collections": []any{}})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.ImportOpenAPI(map[string]any{})
	if err == nil {
		t.Fatal("expected error for empty collections, got nil")
	}
}

func TestImportOpenAPI_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.ImportOpenAPI(map[string]any{})
	if err == nil {
		t.Fatal("expected error for HTTP 401, got nil")
	}
}

// ── GetCollection ─────────────────────────────────────────────────────────────

func TestGetCollection_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"collection": map[string]any{
				"info": map[string]any{
					"name":   "My API",
					"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
				},
				"item": []any{},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	col, err := c.GetCollection("col-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if col.Collection.Info.Name != "My API" {
		t.Errorf("Name = %q, want %q", col.Collection.Info.Name, "My API")
	}
}

func TestGetCollection_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.GetCollection("missing")
	if err == nil {
		t.Fatal("expected error for HTTP 404, got nil")
	}
}

// ── DeleteCollection ──────────────────────────────────────────────────────────

func TestDeleteCollection_Success(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.DeleteCollection("col-id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("DELETE request was not made")
	}
}

func TestDeleteCollection_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.DeleteCollection("col-id")
	if err == nil {
		t.Fatal("expected error for HTTP 403, got nil")
	}
}

// ── UpdateCollection ──────────────────────────────────────────────────────────

func TestUpdateCollection_SendsCorrectBody(t *testing.T) {
	var received CollectionWrapper
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decoding request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	data := &CollectionWrapper{
		Collection: Collection{
			Info:  Info{Name: "My API", Schema: "v2.1.0"},
			Items: []CollectionItem{},
		},
	}
	if err := c.UpdateCollection("col-id", data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received.Collection.Info.Name != "My API" {
		t.Errorf("received name = %q, want %q", received.Collection.Info.Name, "My API")
	}
}

func TestUpdateCollection_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.UpdateCollection("col-id", &CollectionWrapper{})
	if err == nil {
		t.Fatal("expected error for HTTP 400, got nil")
	}
}

// ── CreateCollection ──────────────────────────────────────────────────────────

func TestCreateCollection_SendsCorrectBody(t *testing.T) {
	var received CollectionWrapper
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	data := &CollectionWrapper{
		Collection: Collection{Info: Info{Name: "New API", Schema: "v2.1.0"}},
	}
	if err := c.CreateCollection(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received.Collection.Info.Name != "New API" {
		t.Errorf("received name = %q", received.Collection.Info.Name)
	}
}

func TestCreateCollection_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "conflict", http.StatusConflict)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.CreateCollection(&CollectionWrapper{})
	if err == nil {
		t.Fatal("expected error for HTTP 409, got nil")
	}
}

// ── GetWorkspaceCollections ───────────────────────────────────────────────────

func TestGetWorkspaceCollections_ReturnsMappedCollections(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"workspace": map[string]any{
				"collections": []map[string]any{
					{"id": "id-1", "name": "Collection A"},
					{"id": "id-2", "name": "Collection B"},
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	m, err := c.GetWorkspaceCollections()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m["Collection A"] != "id-1" {
		t.Errorf("Collection A id = %q, want id-1", m["Collection A"])
	}
	if m["Collection B"] != "id-2" {
		t.Errorf("Collection B id = %q, want id-2", m["Collection B"])
	}
}

func TestGetWorkspaceCollections_EmptyWorkspace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"workspace": map[string]any{},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	m, err := c.GetWorkspaceCollections()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestGetWorkspaceCollections_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.GetWorkspaceCollections()
	if err == nil {
		t.Fatal("expected error for HTTP 403, got nil")
	}
}

// ── GetWorkspaceEnvironments ──────────────────────────────────────────────────

func TestGetWorkspaceEnvironments_ReturnsMappedEnvironments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"workspace": map[string]any{
				"environments": []map[string]any{
					{"id": "id-1", "uid": "uid-1", "name": "Production"},
					{"id": "id-2", "uid": "", "name": "Staging"},
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	m, err := c.GetWorkspaceEnvironments()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m["Production"] != "uid-1" {
		t.Errorf("Production id = %q, want uid-1", m["Production"])
	}
	// Falls back to id when uid is empty.
	if m["Staging"] != "id-2" {
		t.Errorf("Staging id = %q, want id-2", m["Staging"])
	}
}

func TestGetWorkspaceEnvironments_EmptyWorkspace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"workspace": map[string]any{},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	m, err := c.GetWorkspaceEnvironments()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestGetWorkspaceEnvironments_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.GetWorkspaceEnvironments()
	if err == nil {
		t.Fatal("expected error for HTTP 403, got nil")
	}
}

// ── GetEnvironment ────────────────────────────────────────────────────────────

func TestGetEnvironment_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"environment": map[string]any{
				"id":   "env-id",
				"name": "Production",
				"values": []map[string]any{
					{"key": "BASE_URL", "value": "https://api.example.com", "enabled": true},
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	env, err := c.GetEnvironment("env-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Environment.Name != "Production" {
		t.Errorf("Name = %q, want %q", env.Environment.Name, "Production")
	}
	if len(env.Environment.Values) != 1 || env.Environment.Values[0].Key != "BASE_URL" {
		t.Errorf("Values = %+v", env.Environment.Values)
	}
}

func TestGetEnvironment_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.GetEnvironment("missing")
	if err == nil {
		t.Fatal("expected error for HTTP 404, got nil")
	}
}

// ── CreateEnvironment ─────────────────────────────────────────────────────────

func TestCreateEnvironment_SendsCorrectBody(t *testing.T) {
	var received EnvironmentWrapper
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	env := &EnvironmentWrapper{
		Environment: Environment{Name: "Staging", Values: []EnvironmentValue{}},
	}
	if err := c.CreateEnvironment(env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received.Environment.Name != "Staging" {
		t.Errorf("received name = %q, want %q", received.Environment.Name, "Staging")
	}
}

func TestCreateEnvironment_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "conflict", http.StatusConflict)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.CreateEnvironment(&EnvironmentWrapper{})
	if err == nil {
		t.Fatal("expected error for HTTP 409, got nil")
	}
}

// ── UpdateEnvironment ─────────────────────────────────────────────────────────

func TestUpdateEnvironment_SendsCorrectBody(t *testing.T) {
	var received EnvironmentWrapper
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	env := &EnvironmentWrapper{
		Environment: Environment{Name: "Production", Values: []EnvironmentValue{}},
	}
	if err := c.UpdateEnvironment("env-id", env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received.Environment.Name != "Production" {
		t.Errorf("received name = %q, want %q", received.Environment.Name, "Production")
	}
}

func TestUpdateEnvironment_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.UpdateEnvironment("env-id", &EnvironmentWrapper{})
	if err == nil {
		t.Fatal("expected error for HTTP 400, got nil")
	}
}

// ── DeleteEnvironment ─────────────────────────────────────────────────────────

func TestDeleteEnvironment_Success(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.DeleteEnvironment("env-id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("DELETE request was not made")
	}
}

func TestDeleteEnvironment_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.DeleteEnvironment("env-id")
	if err == nil {
		t.Fatal("expected error for HTTP 403, got nil")
	}
}

// ── API key header ────────────────────────────────────────────────────────────

func TestClient_SendsAPIKeyHeader(t *testing.T) {
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-API-Key")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("my-secret-key", "ws").withBaseURL(srv.URL)
	_ = c.DeleteCollection("x") // use any method that fires a request

	if gotKey != "my-secret-key" {
		t.Errorf("X-API-Key = %q, want %q", gotKey, "my-secret-key")
	}
}
