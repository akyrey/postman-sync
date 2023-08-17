package postman

import "time"

type Workspace struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Type         string        `json:"type"`
	Description  string        `json:"description"`
	Visibility   string        `json:"visibility"`
	CreatedBy    string        `json:"createdBy"`
	UpdatedBy    string        `json:"updatedBy"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
	Collections  []Collection  `json:"collections"`
	Environments []Environment `json:"environments"`
	Mocks        []Mock        `json:"mocks"`
	Apis         []Api         `json:"apis"`
}
type WorkspaceListResponse struct {
	Workspaces []Workspace `json:"workspaces"`
}
type WorkspaceResponse struct {
	Workspace Workspace `json:"workspace"`
}

type Collection struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	UID  string `json:"uid"`
}

type Environment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	UID  string `json:"uid"`
}

type Mock struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	UID  string `json:"uid"`
}

type Api struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	UID  string `json:"uid"`
}
