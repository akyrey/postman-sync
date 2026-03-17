package postman_test

import (
	"strings"
	"testing"

	"github.com/akyrey/postman-sync/config"
	"github.com/akyrey/postman-sync/postman"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func leafItem(name, method string, path ...string) postman.CollectionItem {
	return postman.CollectionItem{
		Name: name,
		Request: &postman.Request{
			Method: method,
			URL: &postman.URL{
				Raw:      "https://example.com/" + strings.Join(path, "/"),
				Protocol: "https",
				Host:     []string{"example", "com"},
				Path:     path,
			},
		},
	}
}

func folderItem(name string, children ...postman.CollectionItem) postman.CollectionItem {
	return postman.CollectionItem{Name: name, Items: &children}
}

// ── FlattenSingleFolders ──────────────────────────────────────────────────────

func TestFlattenSingleFolders_NoOp_MultipleChildren(t *testing.T) {
	items := []postman.CollectionItem{
		folderItem("Pets",
			leafItem("List pets", "GET", "pets"),
			leafItem("Create pet", "POST", "pets"),
		),
	}
	result := postman.FlattenSingleFolders(items)
	if len(result) != 1 || !result[0].IsFolder() {
		t.Errorf("folder with multiple children should not be flattened, got %+v", result)
	}
}

func TestFlattenSingleFolders_FlattensMatchingName(t *testing.T) {
	inner := leafItem("Get pet", "GET", "pets", "id")
	items := []postman.CollectionItem{
		folderItem("Get pet", inner),
	}
	result := postman.FlattenSingleFolders(items)
	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	if result[0].IsFolder() {
		t.Error("single-child folder matching name should be flattened to leaf")
	}
	if result[0].Name != "Get pet" {
		t.Errorf("Name = %q, want %q", result[0].Name, "Get pet")
	}
}

func TestFlattenSingleFolders_NoFlattenWhenNameDiffers(t *testing.T) {
	inner := leafItem("Get pet", "GET", "pets", "id")
	items := []postman.CollectionItem{
		folderItem("Pets", inner), // folder name != child name
	}
	result := postman.FlattenSingleFolders(items)
	if len(result) != 1 || !result[0].IsFolder() {
		t.Errorf("folder with different name should not be flattened: %+v", result)
	}
}

func TestFlattenSingleFolders_Recursive(t *testing.T) {
	inner := leafItem("Nested", "GET")
	deepFolder := folderItem("Nested", inner)
	outerFolder := folderItem("Outer", deepFolder)
	result := postman.FlattenSingleFolders([]postman.CollectionItem{outerFolder})

	// Outer folder stays (name doesn't match child).
	if len(result) != 1 || !result[0].IsFolder() {
		t.Fatalf("outer folder should remain, got %+v", result)
	}
	children := *result[0].Items
	if len(children) != 1 || children[0].IsFolder() {
		t.Errorf("inner folder should have been flattened into leaf: %+v", children)
	}
}

// ── SortItemsAlpha ────────────────────────────────────────────────────────────

func TestSortItemsAlpha_SortsTopLevel(t *testing.T) {
	items := []postman.CollectionItem{
		leafItem("Zebra", "GET"),
		leafItem("Apple", "GET"),
		leafItem("Mango", "GET"),
	}
	result := postman.SortItemsAlpha(items)
	names := []string{result[0].Name, result[1].Name, result[2].Name}
	want := []string{"Apple", "Mango", "Zebra"}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("item[%d].Name = %q, want %q", i, n, want[i])
		}
	}
}

func TestSortItemsAlpha_SortsRecursively(t *testing.T) {
	folder := folderItem("Folder",
		leafItem("Z endpoint", "GET"),
		leafItem("A endpoint", "GET"),
	)
	result := postman.SortItemsAlpha([]postman.CollectionItem{folder})
	children := *result[0].Items
	if children[0].Name != "A endpoint" || children[1].Name != "Z endpoint" {
		t.Errorf("folder children not sorted: %v, %v", children[0].Name, children[1].Name)
	}
}

func TestSortItemsAlpha_StableWithNoItems(t *testing.T) {
	result := postman.SortItemsAlpha(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

// ── ApplyCommonHeaders ────────────────────────────────────────────────────────

func TestApplyCommonHeaders_InjectsNewHeader(t *testing.T) {
	items := []postman.CollectionItem{leafItem("Get", "GET")}
	headers := []config.Header{{Key: "X-Tenant", Value: "{{tenantId}}", Disabled: false}}
	postman.ApplyCommonHeaders(items, headers)

	got := items[0].Request.Header
	if len(got) != 1 || got[0].Key != "X-Tenant" || got[0].Value != "{{tenantId}}" {
		t.Errorf("headers = %+v", got)
	}
}

func TestApplyCommonHeaders_UpdatesExistingHeader(t *testing.T) {
	item := leafItem("Get", "GET")
	item.Request.Header = []postman.Header{{Key: "X-Tenant", Value: "old"}}
	headers := []config.Header{{Key: "X-Tenant", Value: "new", Disabled: true}}

	postman.ApplyCommonHeaders([]postman.CollectionItem{item}, headers)
	// Note: ApplyCommonHeaders mutates the slice in-place via index, but item was
	// passed by value. Re-read via a pointer-based approach to confirm the index path.
	items := []postman.CollectionItem{item}
	items[0].Request.Header = []postman.Header{{Key: "X-Tenant", Value: "old"}}
	postman.ApplyCommonHeaders(items, headers)

	got := items[0].Request.Header
	if len(got) != 1 || got[0].Value != "new" || !got[0].Disabled {
		t.Errorf("header not updated: %+v", got)
	}
}

func TestApplyCommonHeaders_CaseInsensitiveKeyMatch(t *testing.T) {
	items := []postman.CollectionItem{leafItem("Get", "GET")}
	items[0].Request.Header = []postman.Header{{Key: "x-tenant", Value: "old"}}
	headers := []config.Header{{Key: "X-Tenant", Value: "new"}}
	postman.ApplyCommonHeaders(items, headers)

	got := items[0].Request.Header
	if len(got) != 1 || got[0].Value != "new" {
		t.Errorf("case-insensitive match failed: %+v", got)
	}
}

func TestApplyCommonHeaders_RecursesIntoFolders(t *testing.T) {
	inner := leafItem("Inner", "POST")
	items := []postman.CollectionItem{folderItem("Folder", inner)}
	headers := []config.Header{{Key: "X-App", Value: "myapp"}}
	postman.ApplyCommonHeaders(items, headers)

	children := *items[0].Items
	got := children[0].Request.Header
	if len(got) != 1 || got[0].Key != "X-App" {
		t.Errorf("header not injected into nested request: %+v", got)
	}
}

func TestApplyCommonHeaders_SkipsItemsWithoutRequest(t *testing.T) {
	items := []postman.CollectionItem{{Name: "bare"}} // no Request
	headers := []config.Header{{Key: "X-Test", Value: "v"}}
	postman.ApplyCommonHeaders(items, headers) // must not panic
}

// ── ApplyAuth ─────────────────────────────────────────────────────────────────

func TestApplyAuth_SetsCollectionAuth(t *testing.T) {
	col := &postman.Collection{}
	authCfg := &config.Auth{
		Type:       "bearer",
		Attributes: []config.AuthAttribute{{Key: "token", Value: "{{tok}}"}},
	}
	if err := postman.ApplyAuth(col, authCfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if col.Auth == nil || col.Auth.Type != "bearer" {
		t.Errorf("Auth = %+v", col.Auth)
	}
	if len(col.Auth.Bearer) != 1 || col.Auth.Bearer[0].Key != "token" {
		t.Errorf("Bearer = %+v", col.Auth.Bearer)
	}
}

func TestApplyAuth_NilConfig_NoChange(t *testing.T) {
	col := &postman.Collection{}
	if err := postman.ApplyAuth(col, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if col.Auth != nil {
		t.Errorf("Auth should remain nil when config is nil")
	}
}

func TestApplyAuth_InvalidType_ReturnsError(t *testing.T) {
	col := &postman.Collection{}
	err := postman.ApplyAuth(col, &config.Auth{Type: "unsupported"})
	if err == nil {
		t.Fatal("expected error for unsupported auth type, got nil")
	}
}

// ── ApplyScripts ──────────────────────────────────────────────────────────────

func TestApplyScripts_SetsPrerequest(t *testing.T) {
	col := &postman.Collection{}
	scripts := &config.Scripts{PreRequest: "console.log('pre');"}
	postman.ApplyScripts(col, scripts)

	if len(col.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(col.Events))
	}
	ev := col.Events[0]
	if ev.Listen != "prerequest" {
		t.Errorf("Listen = %q, want prerequest", ev.Listen)
	}
	if len(ev.Script.Exec) == 0 || ev.Script.Exec[0] != "console.log('pre');" {
		t.Errorf("Exec = %v", ev.Script.Exec)
	}
}

func TestApplyScripts_SetsBothEvents(t *testing.T) {
	col := &postman.Collection{}
	scripts := &config.Scripts{
		PreRequest: "// pre",
		Test:       "// test",
	}
	postman.ApplyScripts(col, scripts)

	if len(col.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(col.Events))
	}
	listens := map[string]bool{}
	for _, e := range col.Events {
		listens[e.Listen] = true
	}
	if !listens["prerequest"] || !listens["test"] {
		t.Errorf("events = %+v", col.Events)
	}
}

func TestApplyScripts_PrereqBeforeTest(t *testing.T) {
	col := &postman.Collection{}
	postman.ApplyScripts(col, &config.Scripts{PreRequest: "// pre", Test: "// test"})
	if col.Events[0].Listen != "prerequest" {
		t.Errorf("first event should be prerequest, got %q", col.Events[0].Listen)
	}
	if col.Events[1].Listen != "test" {
		t.Errorf("second event should be test, got %q", col.Events[1].Listen)
	}
}

func TestApplyScripts_NilConfig_NoChange(t *testing.T) {
	col := &postman.Collection{}
	postman.ApplyScripts(col, nil)
	if len(col.Events) != 0 {
		t.Error("Events should remain empty when scripts config is nil")
	}
}

func TestApplyScripts_MultiLineScript(t *testing.T) {
	col := &postman.Collection{}
	postman.ApplyScripts(col, &config.Scripts{Test: "line1\nline2\nline3"})
	exec := col.Events[0].Script.Exec
	if len(exec) != 3 {
		t.Errorf("expected 3 lines, got %d: %v", len(exec), exec)
	}
}

// ── ApplyFolderOverrides ──────────────────────────────────────────────────────

func TestApplyFolderOverrides_SetsAuthOnFolder(t *testing.T) {
	items := []postman.CollectionItem{
		folderItem("Auth", leafItem("Login", "POST", "auth", "login")),
	}
	overrides := map[string]config.FolderOverride{
		"Auth": {Auth: &config.Auth{Type: "noauth"}},
	}
	if err := postman.ApplyFolderOverrides(items, overrides); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if items[0].Auth == nil || items[0].Auth.Type != "noauth" {
		t.Errorf("folder auth = %+v", items[0].Auth)
	}
}

func TestApplyFolderOverrides_SetsScriptsOnFolder(t *testing.T) {
	items := []postman.CollectionItem{
		folderItem("Pets", leafItem("List", "GET")),
	}
	overrides := map[string]config.FolderOverride{
		"Pets": {Scripts: &config.Scripts{Test: "pm.test('ok', ()=>{});"}},
	}
	if err := postman.ApplyFolderOverrides(items, overrides); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items[0].Events) != 1 || items[0].Events[0].Listen != "test" {
		t.Errorf("folder events = %+v", items[0].Events)
	}
}

func TestApplyFolderOverrides_UnknownFolderIsNoop(t *testing.T) {
	items := []postman.CollectionItem{
		folderItem("Pets", leafItem("List", "GET")),
	}
	overrides := map[string]config.FolderOverride{
		"Unknown": {Auth: &config.Auth{Type: "noauth"}},
	}
	if err := postman.ApplyFolderOverrides(items, overrides); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if items[0].Auth != nil {
		t.Error("auth should not be set for unmatched folder")
	}
}

func TestApplyFolderOverrides_SkipsLeaves(t *testing.T) {
	items := []postman.CollectionItem{leafItem("Get", "GET")}
	overrides := map[string]config.FolderOverride{
		"Get": {Auth: &config.Auth{Type: "noauth"}},
	}
	if err := postman.ApplyFolderOverrides(items, overrides); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Leaf items should not have auth set by folder override.
	if items[0].Auth != nil {
		t.Error("leaf items should not be targeted by folder overrides")
	}
}

func TestApplyFolderOverrides_InvalidAuthType(t *testing.T) {
	items := []postman.CollectionItem{folderItem("X", leafItem("Y", "GET"))}
	overrides := map[string]config.FolderOverride{
		"X": {Auth: &config.Auth{Type: "invalid-type"}},
	}
	err := postman.ApplyFolderOverrides(items, overrides)
	if err == nil {
		t.Fatal("expected error for invalid auth type in folder override, got nil")
	}
}

// ── SetBaseURL ────────────────────────────────────────────────────────────────

func TestSetBaseURL_Variable(t *testing.T) {
	items := []postman.CollectionItem{leafItem("Get", "GET", "pets")}
	postman.SetBaseURL(items, "{{baseUrl}}")

	u := items[0].Request.URL
	if len(u.Host) != 1 || u.Host[0] != "{{baseUrl}}" {
		t.Errorf("Host = %v", u.Host)
	}
	if u.Protocol != "" {
		t.Errorf("Protocol should be empty for variable URL, got %q", u.Protocol)
	}
	if !strings.Contains(u.Raw, "{{baseUrl}}") {
		t.Errorf("Raw = %q, should contain {{baseUrl}}", u.Raw)
	}
}

func TestSetBaseURL_LiteralURL(t *testing.T) {
	items := []postman.CollectionItem{leafItem("Get", "GET", "users")}
	postman.SetBaseURL(items, "https://api.example.com")

	u := items[0].Request.URL
	if u.Protocol != "https" {
		t.Errorf("Protocol = %q", u.Protocol)
	}
	want := []string{"api", "example", "com"}
	if len(u.Host) != len(want) {
		t.Fatalf("Host = %v, want %v", u.Host, want)
	}
	for i, part := range want {
		if u.Host[i] != part {
			t.Errorf("Host[%d] = %q, want %q", i, u.Host[i], part)
		}
	}
}

func TestSetBaseURL_RecursesIntoFolders(t *testing.T) {
	inner := leafItem("Inner", "GET", "a")
	items := []postman.CollectionItem{folderItem("Folder", inner)}
	postman.SetBaseURL(items, "{{base}}")

	u := (*items[0].Items)[0].Request.URL
	if len(u.Host) == 0 || u.Host[0] != "{{base}}" {
		t.Errorf("nested URL host = %v", u.Host)
	}
}

func TestSetBaseURL_SkipsNilURL(t *testing.T) {
	items := []postman.CollectionItem{{Name: "bare", Request: &postman.Request{}}}
	postman.SetBaseURL(items, "{{base}}") // must not panic
}

// ── AddDocLinks ───────────────────────────────────────────────────────────────

func TestAddDocLinks_AppendsLink(t *testing.T) {
	items := []postman.CollectionItem{leafItem("Get pet", "GET", "pets", "id")}
	postman.AddDocLinks(items, "https://docs.example.com/#tag/")

	desc := items[0].Request.Description
	if !strings.Contains(desc, "[Docs](") {
		t.Errorf("description missing doc link: %q", desc)
	}
	if !strings.Contains(desc, "https://docs.example.com/#tag/") {
		t.Errorf("description missing base doc URL: %q", desc)
	}
}

func TestAddDocLinks_AppendsToExistingDescription(t *testing.T) {
	items := []postman.CollectionItem{leafItem("Get pet", "GET", "pets", "id")}
	items[0].Request.Description = "existing description"
	postman.AddDocLinks(items, "https://docs.example.com/#tag/")

	desc := items[0].Request.Description
	if !strings.HasPrefix(desc, "existing description") {
		t.Errorf("existing description was overwritten: %q", desc)
	}
	if !strings.Contains(desc, "[Docs](") {
		t.Errorf("doc link not appended: %q", desc)
	}
}

func TestAddDocLinks_SkipsEmptyPath(t *testing.T) {
	item := postman.CollectionItem{
		Name: "Get",
		Request: &postman.Request{
			URL: &postman.URL{Raw: "https://example.com", Path: []string{}},
		},
	}
	postman.AddDocLinks([]postman.CollectionItem{item}, "https://docs.example.com/")
	if item.Request.Description != "" {
		t.Errorf("description should be empty for empty path: %q", item.Request.Description)
	}
}

func TestAddDocLinks_RecursesIntoFolders(t *testing.T) {
	inner := leafItem("List", "GET", "pets")
	items := []postman.CollectionItem{folderItem("Pets", inner)}
	postman.AddDocLinks(items, "https://docs.example.com/#tag/")

	childDesc := (*items[0].Items)[0].Request.Description
	if !strings.Contains(childDesc, "[Docs](") {
		t.Errorf("doc link not added to nested request: %q", childDesc)
	}
}
