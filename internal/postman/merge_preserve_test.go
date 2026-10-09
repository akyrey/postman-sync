package postman_test

import (
	"encoding/json"
	"testing"

	"github.com/akyrey/postman-sync/internal/postman"
)

var preserve = postman.MergeOptions{PreserveRequestData: true}

// decodeRequest builds a Request from JSON so that fields only kept in Extra
// (description, body options, form values) are populated like real API data.
func decodeRequest(t *testing.T, raw string) *postman.Request {
	t.Helper()
	var r postman.Request
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatalf("decoding request: %v", err)
	}
	return &r
}

func mergeOne(t *testing.T, oldReq, newReq string) *postman.Request {
	t.Helper()
	old := []postman.CollectionItem{{Name: "R", Request: decodeRequest(t, oldReq)}}
	new := []postman.CollectionItem{{Name: "R", Request: decodeRequest(t, newReq)}}
	return postman.MergeItemsWith(old, new, preserve)[0].Request
}

func toJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

func headerKeys(hs []postman.Header) []string {
	keys := make([]string, 0, len(hs))
	for _, h := range hs {
		keys = append(keys, h.Key)
	}
	return keys
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ── Headers ───────────────────────────────────────────────────────────────────

func TestMergePreserve_HeaderKeepsOldValueAndDisabledWithNewDescription(t *testing.T) {
	got := mergeOne(t,
		`{"header":[{"key":"x-skp-locale","value":"de-AT","description":"old"}]}`,
		`{"header":[{"key":"x-skp-locale","value":"","description":"Locale (new)","disabled":true}]}`,
	)

	if len(got.Header) != 1 {
		t.Fatalf("headers = %+v", got.Header)
	}
	h := got.Header[0]
	if h.Value != "de-AT" || h.Disabled {
		t.Errorf("expected old value and enabled state, got %+v", h)
	}
	if string(h.Extra["description"]) != `"Locale (new)"` {
		t.Errorf("description should come from spec, got %s", h.Extra["description"])
	}
}

func TestMergePreserve_HeaderMatchIsCaseInsensitive(t *testing.T) {
	got := mergeOne(t,
		`{"header":[{"key":"accept","value":"text/csv"}]}`,
		`{"header":[{"key":"Accept","value":"application/json"}]}`,
	)

	if len(got.Header) != 1 || got.Header[0].Value != "text/csv" || got.Header[0].Key != "Accept" {
		t.Errorf("headers = %+v", got.Header)
	}
}

func TestMergePreserve_HeaderAddedBySpecIsAdded(t *testing.T) {
	got := mergeOne(t,
		`{"header":[{"key":"A","value":"1","description":"a"}]}`,
		`{"header":[{"key":"A","value":"<string>","description":"a"},{"key":"B","value":"<string>","description":"b"}]}`,
	)

	if !equalStrings(headerKeys(got.Header), []string{"A", "B"}) {
		t.Errorf("headers = %+v", got.Header)
	}
}

func TestMergePreserve_HeaderRemovedFromSpecIsDropped(t *testing.T) {
	got := mergeOne(t,
		`{"header":[{"key":"Gone","value":"1","description":"from spec"},{"key":"Kept","value":"2","description":"k"}]}`,
		`{"header":[{"key":"Kept","value":"<string>","description":"k"}]}`,
	)

	if !equalStrings(headerKeys(got.Header), []string{"Kept"}) {
		t.Errorf("headers = %+v", got.Header)
	}
}

func TestMergePreserve_ManuallyAddedHeaderIsKeptAfterSpecHeaders(t *testing.T) {
	got := mergeOne(t,
		`{"header":[{"key":"X-Debug","value":"1"},{"key":"A","value":"a","description":"a"}]}`,
		`{"header":[{"key":"A","value":"<string>","description":"a"}]}`,
	)

	if !equalStrings(headerKeys(got.Header), []string{"A", "X-Debug"}) {
		t.Errorf("headers = %+v", got.Header)
	}
}

// ── Query and path params ─────────────────────────────────────────────────────

func TestMergePreserve_QueryParamsKeepOldValueAndState(t *testing.T) {
	got := mergeOne(t,
		`{"url":{"query":[{"key":"page","value":"3","description":"p"},{"key":"old","value":"x","description":"o"},{"key":"mine","value":"y"}]}}`,
		`{"url":{"query":[{"key":"page","value":"1","description":"Page","disabled":true},{"key":"sort","value":"<string>","description":"s","disabled":true}]}}`,
	)

	q := got.URL.Query
	if len(q) != 3 {
		t.Fatalf("query = %s", toJSON(t, q))
	}
	if *q[0].Key != "page" || *q[0].Value != "3" || q[0].Disabled || string(q[0].Extra["description"]) != `"Page"` {
		t.Errorf("page param = %s", toJSON(t, q[0]))
	}
	if *q[1].Key != "sort" || !q[1].Disabled {
		t.Errorf("sort param = %s", toJSON(t, q[1]))
	}
	if *q[2].Key != "mine" {
		t.Errorf("manual param should be kept last, got %s", toJSON(t, q[2]))
	}
}

func TestMergePreserve_PathVariablesKeepOldValueAndDropUnknown(t *testing.T) {
	got := mergeOne(t,
		`{"url":{"variable":[{"key":"id","value":"42"},{"key":"stale","value":"1"}]}}`,
		`{"url":{"variable":[{"key":"id","value":"<long>","description":"(Required) Id"}]}}`,
	)

	v := got.URL.Variable
	if len(v) != 1 || v[0].Key != "id" || v[0].Value != "42" || string(v[0].Extra["description"]) != `"(Required) Id"` {
		t.Errorf("variables = %s", toJSON(t, v))
	}
}

func TestMergePreserve_URLStructureComesFromSpec(t *testing.T) {
	got := mergeOne(t,
		`{"url":{"raw":"{{baseUrl}}/old","path":["old"]}}`,
		`{"url":{"raw":"{{baseUrl}}/new","path":["new"]}}`,
	)

	if got.URL.Raw != "{{baseUrl}}/new" || got.URL.Path[0] != "new" {
		t.Errorf("url = %s", toJSON(t, got.URL))
	}
}

// ── Body ──────────────────────────────────────────────────────────────────────

func TestMergePreserve_RawJSONBodyMergesFields(t *testing.T) {
	oldBody := `{"name":"Mario","removed":true,"address":{"city":"Rome","zip":"00100"},"tags":["a","b"]}`
	newBody := `{"name":"<string>","address":{"zip":"<string>","country":"<string>"},"tags":["<string>"],"age":"<integer>"}`
	got := mergeOne(t,
		`{"body":{"mode":"raw","raw":`+toJSON(t, oldBody)+`}}`,
		`{"body":{"mode":"raw","raw":`+toJSON(t, newBody)+`,"options":{"raw":{"language":"json"}}}}`,
	)

	want := "{\n" +
		"  \"name\": \"Mario\",\n" +
		"  \"address\": {\n" +
		"    \"zip\": \"00100\",\n" +
		"    \"country\": \"<string>\"\n" +
		"  },\n" +
		"  \"tags\": [\n" +
		"    \"a\",\n" +
		"    \"b\"\n" +
		"  ],\n" +
		"  \"age\": \"<integer>\"\n" +
		"}"
	if got.Body.Raw != want {
		t.Errorf("raw body:\n%s\nwant:\n%s", got.Body.Raw, want)
	}
	if got.Body.Extra["options"] == nil {
		t.Error("body options should come from spec")
	}
}

func TestMergePreserve_RawJSONBodyTypeChangeTakesSpec(t *testing.T) {
	got := mergeOne(t,
		`{"body":{"mode":"raw","raw":"{\"address\":\"Rome\"}"}}`,
		`{"body":{"mode":"raw","raw":"{\"address\":{\"city\":\"<string>\"}}"}}`,
	)

	want := "{\n  \"address\": {\n    \"city\": \"<string>\"\n  }\n}"
	if got.Body.Raw != want {
		t.Errorf("raw body:\n%s\nwant:\n%s", got.Body.Raw, want)
	}
}

func TestMergePreserve_ArrayOfObjectsMergedAgainstSpecTemplate(t *testing.T) {
	got := mergeOne(t,
		`{"body":{"mode":"raw","raw":"{\"items\":[{\"id\":1,\"gone\":0},{\"id\":2}]}"}}`,
		`{"body":{"mode":"raw","raw":"{\"items\":[{\"id\":\"<long>\",\"qty\":\"<integer>\"}]}"}}`,
	)

	want := "{\n  \"items\": [\n" +
		"    {\n      \"id\": 1,\n      \"qty\": \"<integer>\"\n    },\n" +
		"    {\n      \"id\": 2,\n      \"qty\": \"<integer>\"\n    }\n" +
		"  ]\n}"
	if got.Body.Raw != want {
		t.Errorf("raw body:\n%s\nwant:\n%s", got.Body.Raw, want)
	}
}

func TestMergePreserve_NonJSONOldRawBodyIsKept(t *testing.T) {
	oldRaw := `{"id": {{userId}}}`
	got := mergeOne(t,
		`{"body":{"mode":"raw","raw":`+toJSON(t, oldRaw)+`}}`,
		`{"body":{"mode":"raw","raw":"{\"id\":\"<long>\"}"}}`,
	)

	if got.Body.Raw != oldRaw {
		t.Errorf("raw body = %q, want %q", got.Body.Raw, oldRaw)
	}
}

func TestMergePreserve_BodyModeChangeTakesSpec(t *testing.T) {
	got := mergeOne(t,
		`{"body":{"mode":"raw","raw":"{}"}}`,
		`{"body":{"mode":"formdata","formdata":[{"key":"file","type":"file"}]}}`,
	)

	if got.Body.Mode != "formdata" || len(got.Body.FormData) != 1 {
		t.Errorf("body = %s", toJSON(t, got.Body))
	}
}

func TestMergePreserve_BodyRemovedFromSpecIsDropped(t *testing.T) {
	got := mergeOne(t, `{"body":{"mode":"raw","raw":"{}"}}`, `{"method":"GET"}`)

	if got.Body != nil {
		t.Errorf("body should be nil, got %s", toJSON(t, got.Body))
	}
}

func TestMergePreserve_FormDataKeepsOldValuesAndFiles(t *testing.T) {
	got := mergeOne(t,
		`{"body":{"mode":"formdata","formdata":[{"key":"file","type":"file","src":"/tmp/a.png","description":"f"},{"key":"alt","value":"hello","type":"text","disabled":true,"description":"a"}]}}`,
		`{"body":{"mode":"formdata","formdata":[{"key":"file","type":"file","value":null,"description":"(Required) File"},{"key":"alt","value":"<string>","type":"text","description":"Alt"}]}}`,
	)

	f := got.Body.FormData
	if len(f) != 2 {
		t.Fatalf("formdata = %s", toJSON(t, f))
	}
	if string(f[0].Extra["src"]) != `"/tmp/a.png"` || string(f[0].Extra["description"]) != `"(Required) File"` {
		t.Errorf("file field = %s", toJSON(t, f[0]))
	}
	if string(f[1].Extra["value"]) != `"hello"` || !f[1].Disabled {
		t.Errorf("alt field = %s", toJSON(t, f[1]))
	}
}

func TestMergePreserve_URLEncodedKeepsOldValues(t *testing.T) {
	got := mergeOne(t,
		`{"body":{"mode":"urlencoded","urlencoded":[{"key":"grant_type","value":"password","description":"g"}]}}`,
		`{"body":{"mode":"urlencoded","urlencoded":[{"key":"grant_type","value":"<string>","description":"g"}]}}`,
	)

	if string(got.Body.URLEncoded[0].Extra["value"]) != `"password"` {
		t.Errorf("urlencoded = %s", toJSON(t, got.Body.URLEncoded))
	}
}

// ── Options ───────────────────────────────────────────────────────────────────

func TestMergeItems_WithoutPreserveSpecWins(t *testing.T) {
	old := []postman.CollectionItem{{Name: "R", Request: decodeRequest(t, `{"header":[{"key":"A","value":"mine"}]}`)}}
	new := []postman.CollectionItem{{Name: "R", Request: decodeRequest(t, `{"header":[{"key":"A","value":"spec"}]}`)}}

	got := postman.MergeItems(old, new)[0].Request
	if got.Header[0].Value != "spec" {
		t.Errorf("header = %+v", got.Header)
	}
}

func TestMergePreserve_AppliesInsideFolders(t *testing.T) {
	oldLeaf := postman.CollectionItem{Name: "R", Request: decodeRequest(t, `{"header":[{"key":"A","value":"mine"}]}`)}
	newLeaf := postman.CollectionItem{Name: "R", Request: decodeRequest(t, `{"header":[{"key":"A","value":"spec"}]}`)}
	old := []postman.CollectionItem{folderItem("F", oldLeaf)}
	new := []postman.CollectionItem{folderItem("F", newLeaf)}

	got := postman.MergeItemsWith(old, new, preserve)
	if (*got[0].Items)[0].Request.Header[0].Value != "mine" {
		t.Errorf("folder child not merged with preserve: %+v", (*got[0].Items)[0].Request.Header)
	}
}
