package postman_test

import (
	"testing"

	"github.com/akyrey/postman-sync/internal/postman"
)

func TestBuildAuth_Bearer(t *testing.T) {
	attrs := []postman.AuthAttribute{{Key: "token", Value: "{{tok}}", Type: "string"}}
	a, err := postman.BuildAuth("bearer", attrs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Type != "bearer" {
		t.Errorf("Type = %q, want %q", a.Type, "bearer")
	}
	if len(a.Bearer) != 1 || a.Bearer[0].Key != "token" {
		t.Errorf("Bearer = %+v", a.Bearer)
	}
	// Other fields should be empty.
	if len(a.APIKey) != 0 || len(a.Basic) != 0 {
		t.Errorf("unexpected non-empty auth fields")
	}
}

func TestBuildAuth_APIKey(t *testing.T) {
	attrs := []postman.AuthAttribute{{Key: "key", Value: "secret"}}
	a, err := postman.BuildAuth("apikey", attrs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(a.APIKey) != 1 || a.APIKey[0].Key != "key" {
		t.Errorf("APIKey = %+v", a.APIKey)
	}
}

func TestBuildAuth_Basic(t *testing.T) {
	attrs := []postman.AuthAttribute{
		{Key: "username", Value: "user"},
		{Key: "password", Value: "pass"},
	}
	a, err := postman.BuildAuth("basic", attrs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(a.Basic) != 2 {
		t.Errorf("Basic = %+v", a.Basic)
	}
}

func TestBuildAuth_OAuth2(t *testing.T) {
	a, err := postman.BuildAuth("oauth2", []postman.AuthAttribute{{Key: "accessToken", Value: "{{tok}}"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(a.OAuth2) != 1 {
		t.Errorf("OAuth2 = %+v", a.OAuth2)
	}
}

func TestBuildAuth_OAuth1(t *testing.T) {
	a, err := postman.BuildAuth("oauth1", []postman.AuthAttribute{{Key: "consumerKey", Value: "k"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(a.OAuth1) != 1 {
		t.Errorf("OAuth1 = %+v", a.OAuth1)
	}
}

func TestBuildAuth_Digest(t *testing.T) {
	a, err := postman.BuildAuth("digest", []postman.AuthAttribute{{Key: "username", Value: "u"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(a.Digest) != 1 {
		t.Errorf("Digest = %+v", a.Digest)
	}
}

func TestBuildAuth_NTLM(t *testing.T) {
	a, err := postman.BuildAuth("ntlm", []postman.AuthAttribute{{Key: "username", Value: "u"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(a.NTLM) != 1 {
		t.Errorf("NTLM = %+v", a.NTLM)
	}
}

func TestBuildAuth_Hawk(t *testing.T) {
	a, err := postman.BuildAuth("hawk", []postman.AuthAttribute{{Key: "authId", Value: "id"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(a.Hawk) != 1 {
		t.Errorf("Hawk = %+v", a.Hawk)
	}
}

func TestBuildAuth_AWSv4(t *testing.T) {
	a, err := postman.BuildAuth("awsv4", []postman.AuthAttribute{{Key: "accessKey", Value: "k"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(a.AWSv4) != 1 {
		t.Errorf("AWSv4 = %+v", a.AWSv4)
	}
}

func TestBuildAuth_EdgeGrid(t *testing.T) {
	a, err := postman.BuildAuth("edgegrid", []postman.AuthAttribute{{Key: "accessToken", Value: "t"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(a.EdgeGrid) != 1 {
		t.Errorf("EdgeGrid = %+v", a.EdgeGrid)
	}
}

func TestBuildAuth_Noauth(t *testing.T) {
	a, err := postman.BuildAuth("noauth", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Type != "noauth" {
		t.Errorf("Type = %q", a.Type)
	}
}

func TestBuildAuth_UnsupportedType(t *testing.T) {
	_, err := postman.BuildAuth("magic", nil)
	if err == nil {
		t.Fatal("expected error for unsupported auth type, got nil")
	}
}

func TestCollectionItem_IsFolder(t *testing.T) {
	children := []postman.CollectionItem{}
	folder := postman.CollectionItem{Name: "Pets", Items: &children}
	leaf := postman.CollectionItem{Name: "Get Pet", Request: &postman.Request{Method: "GET"}}

	if !folder.IsFolder() {
		t.Error("folder.IsFolder() should be true")
	}
	if leaf.IsFolder() {
		t.Error("leaf.IsFolder() should be false")
	}
}
