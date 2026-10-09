package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/akyrey/postman-sync/internal/config"
	"github.com/akyrey/postman-sync/internal/postman"
)

func TestPrintChanges_ListsChangesAndSummary(t *testing.T) {
	var buf bytes.Buffer
	printChanges(&buf, []postman.ItemChange{
		{Path: "F/New", Kind: postman.ChangeAdded},
		{Path: "F/Edited", Kind: postman.ChangeModified, Details: []string{"method: GET → POST"}},
		{Path: "Old", Kind: postman.ChangeRemoved},
	})

	want := "+ F/New\n" +
		"~ F/Edited\n" +
		"    method: GET → POST\n" +
		"- Old\n" +
		"Summary: 1 added, 1 removed, 1 modified.\n"
	if buf.String() != want {
		t.Errorf("output:\n%s\nwant:\n%s", buf.String(), want)
	}
}

func TestPrintChanges_NoChanges(t *testing.T) {
	var buf bytes.Buffer
	printChanges(&buf, nil)

	if buf.String() != "No changes.\n" {
		t.Errorf("output = %q", buf.String())
	}
}

func TestMergeCollection_DoesNotModifyCurrent(t *testing.T) {
	current := &postman.Collection{
		Info: postman.Info{Name: "API"},
		Items: []postman.CollectionItem{{
			Name:    "R",
			Request: &postman.Request{Header: []postman.Header{{Key: "X-Tenant", Value: "mine"}}},
		}},
	}
	before, _ := json.Marshal(current)
	generated := &postman.Collection{
		Info: postman.Info{Name: "API"},
		Auth: &postman.Auth{Type: "bearer"},
		Items: []postman.CollectionItem{{
			Name:    "R",
			Request: &postman.Request{Header: []postman.Header{{Key: "X-Tenant", Value: "<string>"}}},
		}},
	}
	o := &config.OpenAPIConfig{CommonHeaders: []config.Header{{Key: "X-Tenant", Value: "{{tenant}}", Disabled: true}}}

	updated := mergeCollection(current, generated, o, nil)

	after, _ := json.Marshal(current)
	if !bytes.Equal(before, after) {
		t.Errorf("current was modified:\nbefore: %s\nafter:  %s", before, after)
	}
	h := updated.Collection.Items[0].Request.Header[0]
	if h.Value != "{{tenant}}" || !h.Disabled {
		t.Errorf("common header not applied: %+v", h)
	}
	if updated.Collection.Auth == nil || updated.Collection.Auth.Type != "bearer" {
		t.Errorf("collection auth not applied: %+v", updated.Collection.Auth)
	}
}
