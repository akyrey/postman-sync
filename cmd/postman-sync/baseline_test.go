package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/akyrey/postman-sync/internal/postman"
)

func TestLoadBaseline_MissingReturnsNil(t *testing.T) {
	items, err := loadBaseline(t.TempDir(), "My API")
	if err != nil || items != nil {
		t.Errorf("loadBaseline = %v, %v; want nil, nil", items, err)
	}
}

func TestSaveAndLoadBaseline_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	col := &postman.Collection{
		Info:  postman.Info{Name: "My/API"},
		Items: []postman.CollectionItem{{Name: "Get", Request: &postman.Request{Method: "GET"}}},
	}

	snapshot, err := snapshotBaseline(col)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	col.Items[0].Name = "mutated after snapshot"
	if err := saveBaseline(dir, "My/API", snapshot); err != nil {
		t.Fatalf("save: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "baselines", "My_API.json")); err != nil {
		t.Errorf("baseline file not at sanitized path: %v", err)
	}
	items, err := loadBaseline(dir, "My/API")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(items) != 1 || items[0].Name != "Get" || items[0].Request.Method != "GET" {
		t.Errorf("items = %+v", items)
	}
}

func TestLoadBaseline_CorruptFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	if err := saveBaseline(dir, "My API", []byte("{not json")); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := loadBaseline(dir, "My API"); err == nil {
		t.Error("expected error for corrupt baseline")
	}
}
