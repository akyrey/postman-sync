package postman

import (
	"fmt"
)

// CollectionWrapper is the root envelope returned by the Postman API.
type CollectionWrapper struct {
	Collection Collection `json:"collection"`
}

// Collection is a Postman Collection v2.1.
type Collection struct {
	Info      Info             `json:"info"`
	Items     []CollectionItem `json:"item"`
	Auth      *Auth            `json:"auth,omitempty"`
	Events    []Event          `json:"event,omitempty"`
	Variables []Variable       `json:"variable,omitempty"`
}

// Info holds metadata about the collection.
type Info struct {
	PostmanID string `json:"_postman_id,omitempty"`
	Name      string `json:"name"`
	Schema    string `json:"schema"`
}

// CollectionItem is a union of Item (request leaf) and ItemGroup (folder).
// We distinguish them by the presence of the "item" key (folder) vs "request" key (leaf).
type CollectionItem struct {
	// Leaf request fields
	ID          string     `json:"id,omitempty"`
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	Request     *Request   `json:"request,omitempty"`
	Responses   []any      `json:"response,omitempty"`
	Events      []Event    `json:"event,omitempty"`
	Auth        *Auth      `json:"auth,omitempty"`
	Variables   []Variable `json:"variable,omitempty"`

	// Folder-only field: if non-nil, this item is a folder.
	Items *[]CollectionItem `json:"item,omitempty"`
}

// IsFolder reports whether this CollectionItem is a folder (item-group).
func (ci *CollectionItem) IsFolder() bool {
	return ci.Items != nil
}

// Request represents an HTTP request.
type Request struct {
	Method      string   `json:"method,omitempty"`
	URL         *URL     `json:"url,omitempty"`
	Header      []Header `json:"header,omitempty"`
	Body        *Body    `json:"body,omitempty"`
	Auth        *Auth    `json:"auth,omitempty"`
	Description string   `json:"description,omitempty"`
}

// URL holds the broken-down URL of a request.
type URL struct {
	Raw      string     `json:"raw,omitempty"`
	Protocol string     `json:"protocol,omitempty"`
	Host     []string   `json:"host,omitempty"`
	Path     []string   `json:"path,omitempty"`
	Port     string     `json:"port,omitempty"`
	Query    []URLParam `json:"query,omitempty"`
	Variable []Variable `json:"variable,omitempty"`
}

// URLParam is a query parameter.
type URLParam struct {
	Key      *string `json:"key"`
	Value    *string `json:"value"`
	Disabled bool    `json:"disabled,omitempty"`
}

// Header is a single HTTP header.
type Header struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled,omitempty"`
}

// Body holds request body data.
type Body struct {
	Mode     string `json:"mode,omitempty"`
	Raw      string `json:"raw,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
}

// Auth holds Postman authentication configuration.
type Auth struct {
	Type     string          `json:"type"`
	APIKey   []AuthAttribute `json:"apikey,omitempty"`
	Basic    []AuthAttribute `json:"basic,omitempty"`
	Bearer   []AuthAttribute `json:"bearer,omitempty"`
	OAuth1   []AuthAttribute `json:"oauth1,omitempty"`
	OAuth2   []AuthAttribute `json:"oauth2,omitempty"`
	Digest   []AuthAttribute `json:"digest,omitempty"`
	NTLM     []AuthAttribute `json:"ntlm,omitempty"`
	Hawk     []AuthAttribute `json:"hawk,omitempty"`
	AWSv4    []AuthAttribute `json:"awsv4,omitempty"`
	EdgeGrid []AuthAttribute `json:"edgegrid,omitempty"`
}

// AuthAttribute is a key/value attribute for an auth method.
type AuthAttribute struct {
	Key   string `json:"key"`
	Value any    `json:"value,omitempty"`
	Type  string `json:"type,omitempty"`
}

// Event is a Postman script event (prerequest or test).
type Event struct {
	Listen   string `json:"listen"`
	Script   Script `json:"script"`
	Disabled bool   `json:"disabled,omitempty"`
}

// Script holds a JavaScript snippet.
type Script struct {
	Type string   `json:"type,omitempty"`
	Exec []string `json:"exec"`
	Name string   `json:"name,omitempty"`
}

// Variable is a Postman variable.
type Variable struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
	Type  string `json:"type,omitempty"`
	Name  string `json:"name,omitempty"`
}

// EnvironmentWrapper is the root envelope for Postman environment API responses.
type EnvironmentWrapper struct {
	Environment Environment `json:"environment"`
}

// Environment is a Postman environment with a list of variables.
type Environment struct {
	ID     string             `json:"id,omitempty"`
	Name   string             `json:"name"`
	Values []EnvironmentValue `json:"values"`
}

// EnvironmentValue is a single variable in a Postman environment.
type EnvironmentValue struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Enabled bool   `json:"enabled"`
}

// BuildAuth constructs a postman.Auth from the given type and attributes.
// Attributes are placed under the matching type key.
func BuildAuth(authType string, attrs []AuthAttribute) (*Auth, error) {
	a := &Auth{Type: authType}
	switch authType {
	case "apikey":
		a.APIKey = attrs
	case "basic":
		a.Basic = attrs
	case "bearer":
		a.Bearer = attrs
	case "oauth1":
		a.OAuth1 = attrs
	case "oauth2":
		a.OAuth2 = attrs
	case "digest":
		a.Digest = attrs
	case "ntlm":
		a.NTLM = attrs
	case "hawk":
		a.Hawk = attrs
	case "awsv4":
		a.AWSv4 = attrs
	case "edgegrid":
		a.EdgeGrid = attrs
	case "noauth":
		// no attributes needed
	default:
		return nil, fmt.Errorf("unsupported auth type %q", authType)
	}
	return a, nil
}
