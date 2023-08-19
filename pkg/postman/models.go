package postman

import "time"

/*************************************************************************************************/
/******************************************* WORSPACE ********************************************/
/*************************************************************************************************/
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

/*************************************************************************************************/
/***************************************** COLLECTION ********************************************/
/*************************************************************************************************/
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

// TODO: using anonymous structs at the moment
// Will probably have to change it later on
type Collection struct {
	Info struct {
		PostmanID   string    `json:"_postman_id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Schema      string    `json:"schema"`
		UpdatedAt   time.Time `json:"updatedAt"`
		UID         string    `json:"uid"`
	} `json:"info"`
	Item []struct {
		ID       string     `json:"id"`
		Name     string     `json:"name"`
		UID      string     `json:"uid"`
		Request  Request    `json:"request,omitempty"`
		Response []Response `json:"response,omitempty"`
		Event    []struct {
			Listen string `json:"listen"`
			Script struct {
				ID   string   `json:"id"`
				Type string   `json:"type"`
				Exec []string `json:"exec"`
			} `json:"script"`
		} `json:"event,omitempty"`
		Item []struct {
			Name    string `json:"name"`
			ID      string `json:"id"`
			UID     string `json:"uid"`
			Request struct {
				Method string `json:"method"`
				Header []any  `json:"header"`
				URL    struct {
					Raw  string   `json:"raw"`
					Host []string `json:"host"`
					Path []string `json:"path"`
				} `json:"url"`
			} `json:"request"`
			Response []struct {
				ResponseTime           any    `json:"responseTime"`
				ID                     string `json:"id"`
				Name                   string `json:"name"`
				Status                 string `json:"status"`
				PostmanPreviewlanguage string `json:"_postman_previewlanguage"`
				Body                   string `json:"body"`
				UID                    string `json:"uid"`
				OriginalRequest        struct {
					Method string `json:"method"`
					Header []any  `json:"header"`
					URL    struct {
						Raw  string   `json:"raw"`
						Host []string `json:"host"`
						Path []string `json:"path"`
					} `json:"url"`
				} `json:"originalRequest"`
				Header []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"header"`
				Cookie []any `json:"cookie"`
				Code   int   `json:"code"`
			} `json:"response"`
			ProtocolProfileBehavior struct {
				DisableBodyPruning bool `json:"disableBodyPruning"`
			} `json:"protocolProfileBehavior"`
		} `json:"item,omitempty"`
		ProtocolProfileBehavior struct {
			DisableBodyPruning bool `json:"disableBodyPruning"`
		} `json:"protocolProfileBehavior,omitempty"`
	} `json:"item"`
	Auth struct {
		Type   string `json:"type"`
		Bearer []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
			Type  string `json:"type"`
		} `json:"bearer"`
	} `json:"auth"`
	Event []struct {
		Listen string `json:"listen"`
		Script struct {
			ID   string   `json:"id"`
			Type string   `json:"type"`
			Exec []string `json:"exec"`
		} `json:"script"`
	} `json:"event"`
}
type Request struct {
	Auth struct {
		Type string `json:"type"`
	} `json:"auth"`
	Method string `json:"method"`
	Header []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"header"`
	Body struct {
		Mode string `json:"mode"`
		Raw  string `json:"raw"`
	} `json:"body"`
	URL struct {
		Raw  string   `json:"raw"`
		Host []string `json:"host"`
		Path []string `json:"path"`
	} `json:"url"`
}
type Response struct {
	ResponseTime           any    `json:"responseTime"`
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Status                 string `json:"status"`
	PostmanPreviewlanguage string `json:"_postman_previewlanguage"`
	Body                   string `json:"body"`
	UID                    string `json:"uid"`
	OriginalRequest        struct {
		Method string `json:"method"`
		Header []any  `json:"header"`
		Body   struct {
			Mode string `json:"mode"`
			Raw  string `json:"raw"`
		} `json:"body"`
		URL struct {
			Raw  string   `json:"raw"`
			Host []string `json:"host"`
			Path []string `json:"path"`
		} `json:"url"`
	} `json:"originalRequest"`
	Header []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"header"`
	Cookie []any `json:"cookie"`
	Code   int   `json:"code"`
}

type CollectionResponse struct {
	Collection Collection `json:"collection"`
}

// /*************************************************************************************************/
// /**************************************** ENVIRONMENT ********************************************/
// /*************************************************************************************************/
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
