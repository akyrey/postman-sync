package postman

import "time"

type WorkspaceVisibility string

const (
	Team     WorkspaceVisibility = "team"
	Personal WorkspaceVisibility = "personal"
	Public   WorkspaceVisibility = "public"
)

type WorkspaceInfo struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Type       string              `json:"type"`
	Visibility WorkspaceVisibility `json:"visibility"`
}
type WorkspaceListResponse struct {
	Workspaces []WorkspaceInfo `json:"workspaces"`
}

type Workspace struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Description  string            `json:"description"`
	Visibility   string            `json:"visibility"`
	CreatedBy    string            `json:"createdBy"`
	UpdatedBy    string            `json:"updatedBy"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	Collections  []CollectionInfo  `json:"collections"`
	Environments []EnvironmentInfo `json:"environments"`
	Mocks        []MockInfo        `json:"mocks"`
	Apis         []ApiInfo         `json:"apis"`
}
type WorkspaceResponse struct {
	Workspace Workspace `json:"workspace"`
}

type CollectionInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Owner     string    `json:"owner"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	UID       string    `json:"uid"`
	IsPublic  bool      `json:"isPublic"`
}
type CollectionListResponse struct {
	Collections []CollectionInfo `json:"collections"`
}

type Collection struct {
	Info  Info    `json:"info"`
	Item  []Item  `json:"item"`
	Auth  Auth    `json:"auth"`
	Event []Event `json:"event"`
}
type Info struct {
	PostmanID   string    `json:"_postman_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Schema      string    `json:"schema"`
	UpdatedAt   time.Time `json:"updatedAt"`
	UID         string    `json:"uid"`
}

// TODO: incomplete
type Item struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	Item []Item
}

// TODO: incomplete
type Auth struct {
	Type string `json:"type"`
}

// TODO: incomplete
type Event struct{}

type CollectionResponse struct {
	Collection Collection `json:"collection"`
}

type EnvironmentInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Owner     string    `json:"owner"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	UID       string    `json:"uid"`
	IsPublic  bool      `json:"isPublic"`
}
type EnvironmentListResponse struct {
	Environments []EnvironmentInfo `json:"environments"`
}
type Environment struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Owner     string             `json:"owner"`
	CreatedAt time.Time          `json:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt"`
	Values    []EnvironmentValue `json:"values"`
	IsPublic  bool               `json:"isPublic"`
}
type EnvironmentValue struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}
type EnvironmentResponse struct {
	Environment Environment `json:"environment"`
}

type MockInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	UID  string `json:"uid"`
}

type ApiInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	UID  string `json:"uid"`
}
