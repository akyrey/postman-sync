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
	Type       string              `json:"type,omitempty"`
	Visibility WorkspaceVisibility `json:"visibility,omitempty"`
}
type WorkspaceListResponse struct {
	Workspaces []WorkspaceInfo `json:"workspaces"`
}

type CreateWorkspaceParam struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}
type UpdateWorkspaceParam struct {
	CreateWorkspaceParam
	ID string `json:"id"`
}
type Workspace struct {
	UpdateWorkspaceParam
	Visibility   string            `json:"visibility"`
	CreatedBy    string            `json:"createdBy,omitempty"`
	UpdatedBy    string            `json:"updatedBy,omitempty"`
	CreatedAt    time.Time         `json:"createdAt,omitempty"`
	UpdatedAt    time.Time         `json:"updatedAt,omitempty"`
	Collections  []CollectionInfo  `json:"collections,omitempty"`
	Environments []EnvironmentInfo `json:"environments,omitempty"`
	Mocks        []MockInfo        `json:"mocks,omitempty"`
	Apis         []ApiInfo         `json:"apis,omitempty"`
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

type CreateCollectionInfoParam struct {
	PostmanID   string `json:"_postman_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Schema      string `json:"schema"`
}
type CreateCollectionItemParam struct {
	Name     string                          `json:"name"`
	Request  CollectionRequestParam          `json:"request,omitempty"`
	Response []CreateCollectionResponseParam `json:"response,omitempty"`
	Event    []struct {
		Listen string `json:"listen"`
		Script struct {
			Type string   `json:"type"`
			Exec []string `json:"exec"`
		} `json:"script,omitempty"`
	} `json:"event,omitempty"`
	Item                    []CreateCollectionItemParam `json:"item,omitempty"`
	ProtocolProfileBehavior struct {
		DisableBodyPruning bool `json:"disableBodyPruning"`
	} `json:"protocolProfileBehavior,omitempty"`
}
type CreateCollectionParam struct {
	Info CreateCollectionInfoParam   `json:"info,omitempty"`
	Item []CreateCollectionItemParam `json:"item,omitempty"`
	Auth struct {
		Type   string `json:"type,omitempty"`
		Bearer []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
			Type  string `json:"type"`
		} `json:"bearer,omitempty"`
	} `json:"auth,omitempty"`
	Event []struct {
		Listen string `json:"listen"`
		Script struct {
			Type string   `json:"type"`
			Exec []string `json:"exec"`
		} `json:"script,omitempty"`
	} `json:"event,omitempty"`
}

// TODO: using anonymous structs at the moment
// Will probably have to change it later on
type Collection struct {
	Info struct {
		CreateCollectionInfoParam
		UpdatedAt time.Time `json:"updatedAt"`
		UID       string    `json:"uid"`
	} `json:"info"`
	Item []struct {
		CreateCollectionItemParam
		ID    string `json:"id"`
		UID   string `json:"uid"`
		Event []struct {
			Listen string `json:"listen"`
			Script struct {
				ID   string   `json:"id"`
				Type string   `json:"type"`
				Exec []string `json:"exec"`
			} `json:"script,omitempty"`
		} `json:"event,omitempty"`
	} `json:"item,omitempty"`
	Auth struct {
		Type   string `json:"type"`
		Bearer []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
			Type  string `json:"type"`
		} `json:"bearer,omitempty"`
	} `json:"auth,omitempty"`
	Event []struct {
		Listen string `json:"listen"`
		Script struct {
			ID   string   `json:"id"`
			Type string   `json:"type"`
			Exec []string `json:"exec"`
		} `json:"script,omitempty"`
	} `json:"event,omitempty"`
}
type CollectionRequestParam struct {
	Auth struct {
		Type string `json:"type"`
	} `json:"auth,omitempty"`
	Method string `json:"method"`
	Header []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"header,omitempty"`
	Body struct {
		Mode string `json:"mode,omitempty"`
		Raw  string `json:"raw,omitempty"`
	} `json:"body,omitempty"`
	URL struct {
		Raw  string   `json:"raw"`
		Host []string `json:"host"`
		Path []string `json:"path"`
	} `json:"url,omitempty"`
}
type CreateCollectionResponseParam struct {
	ResponseTime           any    `json:"responseTime"`
	Name                   string `json:"name"`
	Status                 string `json:"status"`
	PostmanPreviewlanguage string `json:"_postman_previewlanguage"`
	Body                   string `json:"body"`
	OriginalRequest        struct {
		Method string `json:"method"`
		Header []any  `json:"header,omitempty"`
		Body   struct {
			Mode string `json:"mode,omitempty"`
			Raw  string `json:"raw,omitempty"`
		} `json:"body,omitempty"`
		URL struct {
			Raw  string   `json:"raw"`
			Host []string `json:"host"`
			Path []string `json:"path"`
		} `json:"url,omitempty"`
	} `json:"originalRequest,omitempty"`
	Header []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"header,omitempty"`
	Cookie []any `json:"cookie,omitempty"`
	Code   int   `json:"code"`
}
type CollectionResponseParam struct {
	CreateCollectionResponseParam
	ID  string `json:"id"`
	UID string `json:"uid"`
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
type CreateEnvironmentParam struct {
	Name     string             `json:"name"`
	Owner    string             `json:"owner"`
	Values   []EnvironmentValue `json:"values"`
	IsPublic bool               `json:"isPublic"`
}
type UpdateEnvironmentParam struct {
	CreateEnvironmentParam
	ID string `json:"id"`
}
type Environment struct {
	UpdateEnvironmentParam
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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
