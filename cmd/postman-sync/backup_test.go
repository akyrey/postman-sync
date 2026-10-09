package main

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/akyrey/postman-sync/internal/postman"
)

func backupWrapper(name string) *postman.CollectionWrapper {
	return &postman.CollectionWrapper{Collection: postman.Collection{
		Info:  postman.Info{Name: name},
		Items: []postman.CollectionItem{{Name: "Get", Request: &postman.Request{Method: "GET"}}},
	}}
}

func listBackupFiles(t *testing.T, stateDir string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(stateDir, "backups", "*", "collections", "*.json"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for i, f := range files {
		rel, _ := filepath.Rel(stateDir, f)
		files[i] = filepath.ToSlash(rel)
	}
	sort.Strings(files)
	return files
}

func TestBackupCollection_WritesImportableFile(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 10, 9, 15, 4, 5, 0, time.UTC)

	path, err := backupCollection(dir, backupWrapper("My/API"), now, 0)
	if err != nil {
		t.Fatalf("backup: %v", err)
	}

	want := filepath.Join(dir, "backups", "20261009T150405Z", "collections", "My_API.json")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	var got postman.CollectionWrapper
	if err := readJSON(path, &got); err != nil {
		t.Fatalf("reading backup: %v", err)
	}
	if got.Collection.Info.Name != "My/API" || len(got.Collection.Items) != 1 {
		t.Errorf("backup content = %+v", got.Collection)
	}
}

func TestBackupCollection_TimestampIsUTC(t *testing.T) {
	dir := t.TempDir()
	rome := time.FixedZone("CEST", 2*60*60)
	now := time.Date(2026, 10, 9, 17, 4, 5, 0, rome)

	path, err := backupCollection(dir, backupWrapper("API"), now, 0)
	if err != nil {
		t.Fatalf("backup: %v", err)
	}
	if filepath.Base(filepath.Dir(filepath.Dir(path))) != "20261009T150405Z" {
		t.Errorf("path = %q, want UTC timestamp directory", path)
	}
}

func TestBackupCollection_KeepsMostRecentPerCollection(t *testing.T) {
	dir := t.TempDir()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 4; i++ {
		if _, err := backupCollection(dir, backupWrapper("API"), start.Add(time.Duration(i)*time.Hour), 2); err != nil {
			t.Fatalf("backup %d: %v", i, err)
		}
	}
	// Another collection's backup must not count towards API's limit.
	if _, err := backupCollection(dir, backupWrapper("Other"), start.Add(10*time.Hour), 2); err != nil {
		t.Fatalf("backup other: %v", err)
	}

	want := []string{
		"backups/20260101T020000Z/collections/API.json",
		"backups/20260101T030000Z/collections/API.json",
		"backups/20260101T100000Z/collections/Other.json",
	}
	got := listBackupFiles(t, dir)
	if len(got) != len(want) {
		t.Fatalf("backups = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("backups = %v, want %v", got, want)
			break
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "backups", "20260101T000000Z")); !os.IsNotExist(err) {
		t.Errorf("pruned backup directory should be removed, stat err = %v", err)
	}
}

func TestBackupCollection_PruningKeepsUnrelatedFiles(t *testing.T) {
	dir := t.TempDir()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := backupCollection(dir, backupWrapper("API"), start, 1); err != nil {
		t.Fatalf("backup: %v", err)
	}
	note := filepath.Join(dir, "backups", "20260101T000000Z", "NOTES.txt")
	if err := os.WriteFile(note, []byte("keep me"), 0o644); err != nil {
		t.Fatalf("write note: %v", err)
	}

	if _, err := backupCollection(dir, backupWrapper("API"), start.Add(time.Hour), 1); err != nil {
		t.Fatalf("backup: %v", err)
	}

	if _, err := os.Stat(note); err != nil {
		t.Errorf("unrelated file should survive pruning: %v", err)
	}
	if got := listBackupFiles(t, dir); len(got) != 1 || got[0] != "backups/20260101T010000Z/collections/API.json" {
		t.Errorf("backups = %v", got)
	}
}

func TestBackupCollection_KeepZeroKeepsAll(t *testing.T) {
	dir := t.TempDir()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		if _, err := backupCollection(dir, backupWrapper("API"), start.Add(time.Duration(i)*time.Hour), 0); err != nil {
			t.Fatalf("backup %d: %v", i, err)
		}
	}

	if got := listBackupFiles(t, dir); len(got) != 3 {
		t.Errorf("backups = %v, want 3", got)
	}
}
