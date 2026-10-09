package postman_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/akyrey/postman-sync/internal/postman"
)

// postmanImportFixture mirrors what Postman's OpenAPI importer produces:
// descriptions carrying "(Required)" and the type, body options, form data,
// header types and other fields not explicitly modelled by our structs.
const postmanImportFixture = `{
  "collection": {
    "info": {
      "_postman_id": "abc",
      "name": "Core API",
      "description": "Collection description",
      "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
      "updatedAt": "2026-01-01T00:00:00.000Z"
    },
    "item": [
      {
        "id": "f1",
        "name": "users",
        "description": "Folder description",
        "item": [
          {
            "id": "r1",
            "name": "Create user",
            "protocolProfileBehavior": {"disableBodyPruning": true},
            "request": {
              "method": "POST",
              "header": [
                {"key": "Content-Type", "value": "application/json"},
                {"key": "x-skp-tenant", "value": "<string>", "description": "(Required) Tenant", "type": "text", "disabled": true}
              ],
              "body": {
                "mode": "raw",
                "raw": "{\n  \"name\": \"<string>\"\n}",
                "options": {"raw": {"headerFamily": "json", "language": "json"}}
              },
              "url": {
                "raw": "{{baseUrl}}/users/:id?limit=<integer>",
                "host": ["{{baseUrl}}"],
                "path": ["users", ":id"],
                "query": [
                  {"key": "limit", "value": "<integer>", "description": "(Required) Page size"},
                  {"key": "sort", "value": "<string>", "description": "Sort order", "disabled": true}
                ],
                "variable": [
                  {"key": "id", "value": "<integer>", "description": "(Required) User id", "type": "any", "disabled": false}
                ]
              }
            },
            "response": [{"name": "OK", "code": 200}]
          },
          {
            "id": "r2",
            "name": "Upload avatar",
            "request": {
              "method": "POST",
              "body": {
                "mode": "formdata",
                "formdata": [
                  {"key": "file", "type": "file", "src": [], "description": "(Required) Avatar"},
                  {"key": "alt", "value": "<string>", "type": "text"}
                ]
              },
              "url": {"raw": "{{baseUrl}}/avatar", "host": ["{{baseUrl}}"], "path": ["avatar"]}
            }
          }
        ]
      }
    ],
    "variable": [{"key": "baseUrl", "value": "/", "type": "string"}]
  }
}`

func TestCollectionWrapper_RoundTripIsLossless(t *testing.T) {
	var wrapper postman.CollectionWrapper
	if err := json.Unmarshal([]byte(postmanImportFixture), &wrapper); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	out, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var want, got any
	if err := json.Unmarshal([]byte(postmanImportFixture), &want); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("round trip lost data\nwant: %s\ngot:  %s", postmanImportFixture, out)
	}
}

func TestCollectionWrapper_TypedFieldEditsWinOverExtras(t *testing.T) {
	var wrapper postman.CollectionWrapper
	if err := json.Unmarshal([]byte(postmanImportFixture), &wrapper); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	req := (*wrapper.Collection.Items[0].Items)[0].Request
	req.Header[1].Disabled = false
	req.Header[1].Value = "{{domain}}"

	out, err := json.Marshal(req.Header[1])
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, present := got["disabled"]; present {
		t.Errorf("disabled should be omitted once cleared, got %s", out)
	}
	if got["value"] != "{{domain}}" || got["description"] != "(Required) Tenant" || got["type"] != "text" {
		t.Errorf("unexpected header JSON: %s", out)
	}
}
