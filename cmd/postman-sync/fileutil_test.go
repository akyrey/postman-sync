package main

import (
	"os"
	"path/filepath"
	"testing"
)

// ── sanitizeFilename ──────────────────────────────────────────────────────────

func TestSanitizeFilename_ReplacesUnsafeChars(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"My API", "My API"},
		{"foo/bar", "foo_bar"},
		{`foo\bar`, "foo_bar"},
		{"foo:bar", "foo_bar"},
		{"foo*bar", "foo_bar"},
		{"foo?bar", "foo_bar"},
		{`foo"bar`, "foo_bar"},
		{"foo<bar", "foo_bar"},
		{"foo>bar", "foo_bar"},
		{"foo|bar", "foo_bar"},
		{"all:safe*chars?replaced", "all_safe_chars_replaced"},
		{"normal-name_v2", "normal-name_v2"},
	}
	for _, tc := range cases {
		got := sanitizeFilename(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ── writeJSON / readJSON ──────────────────────────────────────────────────────

func TestWriteJSON_Pretty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	data := map[string]any{"name": "test", "value": 42}
	if err := writeJSON(path, data, true); err != nil {
		t.Fatalf("writeJSON: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	// Pretty output should contain newlines and indentation.
	content := string(b)
	if content[0] != '{' {
		t.Errorf("expected JSON object, got: %s", content[:10])
	}
	// Should contain a newline (pretty-printed).
	found := false
	for _, c := range content {
		if c == '\n' {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected pretty-printed JSON with newlines, got compact output")
	}
}

func TestWriteJSON_Compact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	data := map[string]string{"key": "val"}
	if err := writeJSON(path, data, false); err != nil {
		t.Fatalf("writeJSON: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	content := string(b)
	// Compact output should not contain newlines.
	for _, c := range content {
		if c == '\n' {
			t.Errorf("expected compact JSON without newlines, got: %s", content)
			break
		}
	}
}

func TestReadJSON_RoundTrip(t *testing.T) {
	type payload struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")

	original := payload{Name: "hello", Count: 7}
	if err := writeJSON(path, original, false); err != nil {
		t.Fatalf("writeJSON: %v", err)
	}

	var got payload
	if err := readJSON(path, &got); err != nil {
		t.Fatalf("readJSON: %v", err)
	}
	if got != original {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, original)
	}
}

func TestReadJSON_FileNotFound(t *testing.T) {
	err := readJSON(filepath.Join(t.TempDir(), "missing.json"), &struct{}{})
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestReadJSON_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte(`{bad`), 0o644); err != nil {
		t.Fatalf("writing file: %v", err)
	}
	err := readJSON(path, &struct{}{})
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// ── listEntityFiles ───────────────────────────────────────────────────────────

func TestListEntityFiles_ReturnsJSONFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.json", "b.json", "c.txt", "d.yaml"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	files, err := listEntityFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 .json files, got %d: %v", len(files), files)
	}
	for _, f := range files {
		if filepath.Ext(f) != ".json" {
			t.Errorf("non-json file in result: %s", f)
		}
	}
}

func TestListEntityFiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	files, err := listEntityFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected empty result, got %v", files)
	}
}

func TestListEntityFiles_DirNotFound(t *testing.T) {
	_, err := listEntityFiles(filepath.Join(t.TempDir(), "nonexistent"))
	if err == nil {
		t.Fatal("expected error for missing directory, got nil")
	}
}

func TestListEntityFiles_SkipsSubdirs(t *testing.T) {
	dir := t.TempDir()
	// Create a subdirectory named with .json (edge case).
	subdir := filepath.Join(dir, "sub.json")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("creating subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "real.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("writing file: %v", err)
	}

	files, err := listEntityFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file (subdir should be skipped), got %d: %v", len(files), files)
	}
}

// ── shouldImport ──────────────────────────────────────────────────────────────

func TestShouldImport_All(t *testing.T) {
	if !shouldImport("anything", []string{"all"}) {
		t.Error("expected true for 'all' selection")
	}
}

func TestShouldImport_ExactMatch(t *testing.T) {
	if !shouldImport("My API", []string{"Other", "My API"}) {
		t.Error("expected true for exact name match")
	}
}

func TestShouldImport_NoMatch(t *testing.T) {
	if shouldImport("My API", []string{"Other", "Another"}) {
		t.Error("expected false when name not in selection")
	}
}

func TestShouldImport_EmptySelection(t *testing.T) {
	if shouldImport("My API", []string{}) {
		t.Error("expected false for empty selection")
	}
}
