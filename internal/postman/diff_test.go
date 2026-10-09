package postman_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/akyrey/postman-sync/internal/postman"
)

func leafJSON(t *testing.T, name, request string) postman.CollectionItem {
	t.Helper()
	return postman.CollectionItem{Name: name, Request: decodeRequest(t, request)}
}

func TestDiffItems_NoChanges(t *testing.T) {
	items := []postman.CollectionItem{folderItem("F", leafJSON(t, "R", `{"method":"GET"}`))}

	if got := postman.DiffItems(items, items); len(got) != 0 {
		t.Errorf("expected no changes, got %+v", got)
	}
}

func TestDiffItems_AddedAndRemovedRequests(t *testing.T) {
	old := []postman.CollectionItem{folderItem("F", leafJSON(t, "Old", `{}`), leafJSON(t, "Kept", `{}`))}
	new := []postman.CollectionItem{folderItem("F", leafJSON(t, "Kept", `{}`), leafJSON(t, "New", `{}`))}

	got := postman.DiffItems(old, new)
	want := []postman.ItemChange{
		{Path: "F/New", Kind: postman.ChangeAdded},
		{Path: "F/Old", Kind: postman.ChangeRemoved},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestDiffItems_AddedFolderReportsFolderOnly(t *testing.T) {
	new := []postman.CollectionItem{folderItem("F", leafJSON(t, "R", `{}`))}

	got := postman.DiffItems(nil, new)
	want := []postman.ItemChange{{Path: "F", Kind: postman.ChangeAdded, Details: []string{"folder with 1 request(s)"}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestDiffItems_RequestDetails(t *testing.T) {
	old := []postman.CollectionItem{leafJSON(t, "R", `{
		"method":"GET",
		"url":{"raw":"{{baseUrl}}/a","query":[{"key":"page","value":"1"},{"key":"gone","value":"x"}],"variable":[{"key":"id","value":"1"}]},
		"header":[{"key":"A","value":"1"},{"key":"B","value":"b","description":"old"}],
		"body":{"mode":"raw","raw":"{}"}
	}`)}
	new := []postman.CollectionItem{leafJSON(t, "R", `{
		"method":"POST",
		"url":{"raw":"{{baseUrl}}/b","query":[{"key":"page","value":"1","disabled":true},{"key":"size","value":"10"}],"variable":[{"key":"id","value":"2"}]},
		"header":[{"key":"A","value":"2"},{"key":"B","value":"b","description":"new"}],
		"body":{"mode":"raw","raw":"{\"a\":1}"}
	}`)}

	got := postman.DiffItems(old, new)
	want := []postman.ItemChange{{Path: "R", Kind: postman.ChangeModified, Details: []string{
		`method: GET → POST`,
		`url: "{{baseUrl}}/a" → "{{baseUrl}}/b"`,
		`header A: value "1" → "2"`,
		`header B: description changed`,
		`query page: disabled false → true`,
		`query size: added`,
		`query gone: removed`,
		`path variable id: value "1" → "2"`,
		`body: raw content changed`,
	}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got:\n%+v\nwant:\n%+v", got, want)
	}
}

func TestDiffItems_BodyModeAndFormFields(t *testing.T) {
	old := []postman.CollectionItem{
		leafJSON(t, "Mode", `{"body":{"mode":"raw","raw":"{}"}}`),
		leafJSON(t, "Form", `{"body":{"mode":"formdata","formdata":[{"key":"f","value":"a","type":"text"}]}}`),
	}
	new := []postman.CollectionItem{
		leafJSON(t, "Mode", `{"body":{"mode":"formdata"}}`),
		leafJSON(t, "Form", `{"body":{"mode":"formdata","formdata":[{"key":"f","value":"b","type":"text"}]}}`),
	}

	got := postman.DiffItems(old, new)
	want := []postman.ItemChange{
		{Path: "Mode", Kind: postman.ChangeModified, Details: []string{`body: mode "raw" → "formdata"`}},
		{Path: "Form", Kind: postman.ChangeModified, Details: []string{`form field f: value "a" → "b"`}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got:\n%+v\nwant:\n%+v", got, want)
	}
}

func TestDiffItems_AuthScriptsAndResponses(t *testing.T) {
	old := []postman.CollectionItem{{Name: "R", Request: &postman.Request{}, Auth: &postman.Auth{Type: "bearer"}}}
	new := []postman.CollectionItem{{
		Name:      "R",
		Request:   &postman.Request{},
		Events:    []postman.Event{{Listen: "test", Script: postman.Script{Exec: []string{"x"}}}},
		Responses: []any{map[string]any{"name": "OK"}},
	}}

	got := postman.DiffItems(old, new)
	want := []postman.ItemChange{{Path: "R", Kind: postman.ChangeModified, Details: []string{
		"auth: bearer → inherit", "scripts changed", "saved responses: 0 → 1",
	}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got:\n%+v\nwant:\n%+v", got, want)
	}
}

func TestDiffItems_FolderAuthChange(t *testing.T) {
	oldFolder := folderItem("F", leafJSON(t, "R", `{}`))
	newFolder := folderItem("F", leafJSON(t, "R", `{}`))
	newFolder.Auth = &postman.Auth{Type: "noauth"}

	got := postman.DiffItems([]postman.CollectionItem{oldFolder}, []postman.CollectionItem{newFolder})
	want := []postman.ItemChange{{Path: "F", Kind: postman.ChangeModified, Details: []string{"auth: inherit → noauth"}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestDiffItems_LongValuesAreTruncated(t *testing.T) {
	long := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"
	old := []postman.CollectionItem{leafJSON(t, "R", `{"header":[{"key":"A","value":"`+long+`"}]}`)}
	new := []postman.CollectionItem{leafJSON(t, "R", `{"header":[{"key":"A","value":"x"}]}`)}

	got := postman.DiffItems(old, new)[0].Details[0]
	want := `header A: value "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefgh…" → "x"`
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestDiffCollections_CollectionLevelChanges(t *testing.T) {
	old := &postman.Collection{Auth: &postman.Auth{Type: "bearer"}}
	new := &postman.Collection{
		Auth:   &postman.Auth{Type: "oauth2"},
		Events: []postman.Event{{Listen: "prerequest"}},
		Items:  []postman.CollectionItem{leafJSON(t, "R", `{}`)},
	}

	got := postman.DiffCollections(old, new)
	want := []postman.ItemChange{
		{Path: "(collection)", Kind: postman.ChangeModified, Details: []string{"auth: bearer → oauth2", "scripts changed"}},
		{Path: "R", Kind: postman.ChangeAdded},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestDiffItems_IgnoresJSONEscapingDifferences(t *testing.T) {
	item := func(value, description string) postman.CollectionItem {
		return postman.CollectionItem{Name: "R", Request: &postman.Request{
			Header: []postman.Header{{Key: "A", Extra: postman.Extra{"description": json.RawMessage(description)}}},
			Body: &postman.Body{Mode: "formdata", FormData: []postman.FormParam{
				{Key: "f", Extra: postman.Extra{"value": json.RawMessage(value)}},
			}},
		}}
	}
	old := []postman.CollectionItem{item(`"\u003cenum\u003e"`, `"\u003cb\u003e"`)}
	new := []postman.CollectionItem{item(`"<enum>"`, `"<b>"`)}

	if got := postman.DiffItems(old, new); len(got) != 0 {
		t.Errorf("expected no changes, got %+v", got)
	}
}
