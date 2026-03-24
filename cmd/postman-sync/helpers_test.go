package main

import (
	"os"
	"testing"
)

// ── splitCSV ──────────────────────────────────────────────────────────────────

func TestSplitCSV_Single(t *testing.T) {
	got := splitCSV("all")
	if len(got) != 1 || got[0] != "all" {
		t.Errorf("splitCSV(%q) = %v, want [all]", "all", got)
	}
}

func TestSplitCSV_Multiple(t *testing.T) {
	got := splitCSV("My API,Other API,Third")
	if len(got) != 3 {
		t.Fatalf("splitCSV: len = %d, want 3: %v", len(got), got)
	}
	if got[0] != "My API" || got[1] != "Other API" || got[2] != "Third" {
		t.Errorf("splitCSV result = %v", got)
	}
}

func TestSplitCSV_TrimsSpaces(t *testing.T) {
	got := splitCSV(" foo , bar , baz ")
	if len(got) != 3 || got[0] != "foo" || got[1] != "bar" || got[2] != "baz" {
		t.Errorf("splitCSV result = %v, want [foo bar baz]", got)
	}
}

func TestSplitCSV_Empty(t *testing.T) {
	got := splitCSV("")
	if got != nil {
		t.Errorf("splitCSV(%q) = %v, want nil", "", got)
	}
}

func TestSplitCSV_SkipsEmptyParts(t *testing.T) {
	got := splitCSV("a,,b")
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("splitCSV result = %v, want [a b]", got)
	}
}

// ── resolveEntities ───────────────────────────────────────────────────────────

func TestResolveEntities_All(t *testing.T) {
	available := map[string]string{"A": "id-1", "B": "id-2"}
	result := resolveEntities([]string{"all"}, available)
	if len(result) != 2 {
		t.Errorf("expected all 2 entities, got %d", len(result))
	}
	if result["A"] != "id-1" || result["B"] != "id-2" {
		t.Errorf("result = %v", result)
	}
}

func TestResolveEntities_Specific(t *testing.T) {
	available := map[string]string{"A": "id-1", "B": "id-2", "C": "id-3"}
	result := resolveEntities([]string{"A", "C"}, available)
	if len(result) != 2 {
		t.Fatalf("expected 2 entities, got %d: %v", len(result), result)
	}
	if result["A"] != "id-1" || result["C"] != "id-3" {
		t.Errorf("result = %v", result)
	}
}

func TestResolveEntities_UnknownNameWarns(t *testing.T) {
	// Redirect stderr to capture warning.
	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	available := map[string]string{"A": "id-1"}
	result := resolveEntities([]string{"A", "Unknown"}, available)

	_ = w.Close()
	os.Stderr = origStderr
	buf := make([]byte, 256)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if len(result) != 1 {
		t.Errorf("expected 1 matched entity, got %d", len(result))
	}
	if output == "" {
		t.Error("expected a warning on stderr for unknown entity, got none")
	}
}

func TestResolveEntities_EmptySelection(t *testing.T) {
	available := map[string]string{"A": "id-1"}
	result := resolveEntities([]string{}, available)
	if len(result) != 0 {
		t.Errorf("expected empty result for empty selection, got %v", result)
	}
}

func TestResolveEntities_AllWithOtherTerms(t *testing.T) {
	// "all" anywhere in the slice should return everything.
	available := map[string]string{"A": "id-1", "B": "id-2"}
	result := resolveEntities([]string{"A", "all"}, available)
	if len(result) != 2 {
		t.Errorf("expected all 2 entities when 'all' present, got %d", len(result))
	}
}
