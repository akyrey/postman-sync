package postman_test

import (
	"testing"

	"github.com/akyrey/postman-sync/postman"
)

// ── MergeItems ────────────────────────────────────────────────────────────────

func TestMergeItems_NewItemAdded(t *testing.T) {
	old := []postman.CollectionItem{}
	new := []postman.CollectionItem{leafItem("New endpoint", "GET")}

	result := postman.MergeItems(old, new)
	if len(result) != 1 || result[0].Name != "New endpoint" {
		t.Errorf("expected new item to be added, got %+v", result)
	}
}

func TestMergeItems_NewItemHasIDStripped(t *testing.T) {
	old := []postman.CollectionItem{}
	n := leafItem("Get", "GET")
	n.ID = "postman-generated-id"

	result := postman.MergeItems(old, []postman.CollectionItem{n})
	if result[0].ID != "" {
		t.Errorf("ID should be stripped for new items, got %q", result[0].ID)
	}
}

func TestMergeItems_RemovedItemDropped(t *testing.T) {
	old := []postman.CollectionItem{
		leafItem("Old endpoint", "DELETE"),
		leafItem("Kept endpoint", "GET"),
	}
	new := []postman.CollectionItem{leafItem("Kept endpoint", "GET")}

	result := postman.MergeItems(old, new)
	if len(result) != 1 || result[0].Name != "Kept endpoint" {
		t.Errorf("removed item should be dropped, got %+v", result)
	}
}

func TestMergeItems_LeafPreservesOldAuth(t *testing.T) {
	oldAuth := &postman.Auth{Type: "bearer", Bearer: []postman.AuthAttribute{{Key: "token", Value: "{{tok}}"}}}
	old := []postman.CollectionItem{{
		Name:    "Get pet",
		Request: &postman.Request{Method: "GET"},
		Auth:    oldAuth,
	}}
	new := []postman.CollectionItem{leafItem("Get pet", "GET", "pets", "id")}

	result := postman.MergeItems(old, new)
	if result[0].Auth == nil || result[0].Auth.Type != "bearer" {
		t.Errorf("auth should be preserved from old: %+v", result[0].Auth)
	}
}

func TestMergeItems_LeafPreservesOldEvents(t *testing.T) {
	oldEvents := []postman.Event{{
		Listen: "test",
		Script: postman.Script{Exec: []string{"pm.test('ok', ()=>{});"}},
	}}
	old := []postman.CollectionItem{{
		Name:    "Get pet",
		Request: &postman.Request{Method: "GET"},
		Events:  oldEvents,
	}}
	new := []postman.CollectionItem{leafItem("Get pet", "GET", "pets", "id")}

	result := postman.MergeItems(old, new)
	if len(result[0].Events) == 0 || result[0].Events[0].Listen != "test" {
		t.Errorf("events should be preserved from old: %+v", result[0].Events)
	}
}

func TestMergeItems_LeafPreservesOldResponses(t *testing.T) {
	old := []postman.CollectionItem{{
		Name:      "Get pet",
		Request:   &postman.Request{Method: "GET"},
		Responses: []any{map[string]any{"name": "200 OK"}},
	}}
	new := []postman.CollectionItem{leafItem("Get pet", "GET", "pets", "id")}

	result := postman.MergeItems(old, new)
	if len(result[0].Responses) == 0 {
		t.Error("saved responses should be preserved from old")
	}
}

func TestMergeItems_LeafUpdatesRequest(t *testing.T) {
	old := []postman.CollectionItem{{
		Name: "Get pet",
		Request: &postman.Request{
			Method: "GET",
			URL:    &postman.URL{Raw: "https://old.example.com/pets"},
		},
	}}
	new := []postman.CollectionItem{leafItem("Get pet", "GET", "pets", "petId")}

	result := postman.MergeItems(old, new)
	// URL should come from new.
	if result[0].Request.URL.Raw != new[0].Request.URL.Raw {
		t.Errorf("URL not updated from new: %q", result[0].Request.URL.Raw)
	}
}

func TestMergeItems_LeafPreservesOldID(t *testing.T) {
	old := []postman.CollectionItem{{
		Name:    "Get pet",
		ID:      "old-internal-id",
		Request: &postman.Request{Method: "GET"},
	}}
	new := []postman.CollectionItem{leafItem("Get pet", "GET")}

	result := postman.MergeItems(old, new)
	if result[0].ID != "old-internal-id" {
		t.Errorf("ID should be preserved: %q", result[0].ID)
	}
}

func TestMergeItems_LeafNewAuthNotOverriddenByNilOld(t *testing.T) {
	// Old item has no auth, new has auth set from config transformation.
	newAuth := &postman.Auth{Type: "noauth"}
	new := []postman.CollectionItem{{
		Name:    "Get pet",
		Request: &postman.Request{Method: "GET"},
		Auth:    newAuth,
	}}
	old := []postman.CollectionItem{{
		Name:    "Get pet",
		Request: &postman.Request{Method: "GET"},
		Auth:    nil,
	}}
	result := postman.MergeItems(old, new)
	// Old auth is nil, so new auth should remain.
	if result[0].Auth == nil || result[0].Auth.Type != "noauth" {
		t.Errorf("new auth should be kept when old has none: %+v", result[0].Auth)
	}
}

// ── Folder merging ────────────────────────────────────────────────────────────

func TestMergeItems_FolderChildrenMerged(t *testing.T) {
	oldFolder := folderItem("Pets",
		leafItem("List pets", "GET"),
		leafItem("Old endpoint", "DELETE"),
	)
	newFolder := folderItem("Pets",
		leafItem("List pets", "GET"),
		leafItem("New endpoint", "POST"),
	)

	result := postman.MergeItems([]postman.CollectionItem{oldFolder}, []postman.CollectionItem{newFolder})
	if len(result) != 1 || !result[0].IsFolder() {
		t.Fatalf("expected one merged folder, got %+v", result)
	}
	children := *result[0].Items
	if len(children) != 2 {
		t.Fatalf("expected 2 children (List pets + New endpoint), got %d: %+v", len(children), children)
	}
	names := map[string]bool{}
	for _, c := range children {
		names[c.Name] = true
	}
	if !names["List pets"] {
		t.Error("'List pets' should be present")
	}
	if !names["New endpoint"] {
		t.Error("'New endpoint' should be present (new item)")
	}
	if names["Old endpoint"] {
		t.Error("'Old endpoint' should be removed (not in new spec)")
	}
}

func TestMergeItems_FolderPreservesOldAuth(t *testing.T) {
	oldAuth := &postman.Auth{Type: "bearer"}
	oldFolder := folderItem("Admin", leafItem("List", "GET"))
	oldFolder.Auth = oldAuth

	newFolder := folderItem("Admin", leafItem("List", "GET"))
	// New folder has no auth (config ran first time on old, manual tweak since).

	result := postman.MergeItems(
		[]postman.CollectionItem{oldFolder},
		[]postman.CollectionItem{newFolder},
	)
	if result[0].Auth == nil || result[0].Auth.Type != "bearer" {
		t.Errorf("folder auth should be preserved from old: %+v", result[0].Auth)
	}
}

func TestMergeItems_FolderNewAuthNotOverridden(t *testing.T) {
	// If new folder already has auth (from config transform), keep it.
	newAuth := &postman.Auth{Type: "noauth"}
	newFolder := folderItem("Public", leafItem("List", "GET"))
	newFolder.Auth = newAuth

	oldFolder := folderItem("Public", leafItem("List", "GET"))
	// Old has no auth.

	result := postman.MergeItems(
		[]postman.CollectionItem{oldFolder},
		[]postman.CollectionItem{newFolder},
	)
	if result[0].Auth == nil || result[0].Auth.Type != "noauth" {
		t.Errorf("new folder auth should be kept when old has none: %+v", result[0].Auth)
	}
}

func TestMergeItems_FolderPreservesOldEvents(t *testing.T) {
	oldEvents := []postman.Event{{Listen: "prerequest", Script: postman.Script{Exec: []string{"// pre"}}}}
	oldFolder := folderItem("Pets", leafItem("List", "GET"))
	oldFolder.Events = oldEvents

	newFolder := folderItem("Pets", leafItem("List", "GET"))

	result := postman.MergeItems(
		[]postman.CollectionItem{oldFolder},
		[]postman.CollectionItem{newFolder},
	)
	if len(result[0].Events) == 0 || result[0].Events[0].Listen != "prerequest" {
		t.Errorf("folder events should be preserved: %+v", result[0].Events)
	}
}

func TestMergeItems_EmptyOld(t *testing.T) {
	new := []postman.CollectionItem{
		leafItem("A", "GET"),
		leafItem("B", "POST"),
	}
	result := postman.MergeItems(nil, new)
	if len(result) != 2 {
		t.Errorf("expected 2 items from empty old, got %d", len(result))
	}
}

func TestMergeItems_EmptyNew(t *testing.T) {
	old := []postman.CollectionItem{leafItem("A", "GET")}
	result := postman.MergeItems(old, nil)
	if len(result) != 0 {
		t.Errorf("expected 0 items when new is empty (spec is source of truth), got %d", len(result))
	}
}

func TestMergeItems_OrderFollowsNew(t *testing.T) {
	// Merge result should follow new's order (which is alphabetically sorted).
	old := []postman.CollectionItem{
		leafItem("Zebra", "GET"),
		leafItem("Apple", "GET"),
	}
	new := []postman.CollectionItem{
		leafItem("Apple", "GET"),
		leafItem("Zebra", "GET"),
	}
	result := postman.MergeItems(old, new)
	if result[0].Name != "Apple" || result[1].Name != "Zebra" {
		t.Errorf("order should follow new: got %q, %q", result[0].Name, result[1].Name)
	}
}
