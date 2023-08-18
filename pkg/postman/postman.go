package postman

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	url             = "https://api.getpostman.com/"
	apiKeyHeader    = "x-api-key"
	contentType     = "Content-Type"
	contentTypeJSON = "application/json"
)

var client *http.Client

func init() {
	// TODO: add options from env?
	client = httpClientWithFallBack(nil)
}

type Postman struct {
	ApiKey string
}

/*************************************************************************************************/
/******************************************* WORSPACE ********************************************/
/*************************************************************************************************/
func (p *Postman) RetrieveWorkspaces() ([]WorkspaceInfo, error) {
	body, err := p.performRequest("GET", fmt.Sprintf("%sworkspaces/", url), nil)
	if err != nil {
		return nil, err
	}

	defer body.Close()
	var resp WorkspaceListResponse
	err = json.NewDecoder(body).Decode(&resp)
	if err != nil {
		return nil, err
	}

	return resp.Workspaces, nil
}

func (p *Postman) RetrieveWorkspace(id string) (*Workspace, error) {
	body, err := p.performRequest("GET", fmt.Sprintf("%sworkspaces/%s", url, id), nil)
	if err != nil {
		return nil, err
	}

	defer body.Close()
	var resp WorkspaceResponse
	err = json.NewDecoder(body).Decode(&resp)
	if err != nil {
		return nil, err
	}

	return &resp.Workspace, nil
}

func (p *Postman) CreateWorkspace(ws Workspace) (*Workspace, error) {
	reqBody, err := json.Marshal(WorkspaceResponse{Workspace: ws})
	if err != nil {
		return nil, err
	}
	body, err := p.performRequest("POST", fmt.Sprintf("%sworkspaces", url), reqBody)
	if err != nil {
		return nil, err
	}

	defer body.Close()
	var workspace Workspace
	err = json.NewDecoder(body).Decode(&workspace)
	if err != nil {
		return nil, err
	}

	return &workspace, nil
}

func (p *Postman) UpdateWorkspace(uid string, ws Workspace) (*Workspace, error) {
	reqBody, err := json.Marshal(WorkspaceResponse{Workspace: ws})
	if err != nil {
		return nil, err
	}
	body, err := p.performRequest("PUT", fmt.Sprintf("%sworkspaces/%s", url, uid), reqBody)
	if err != nil {
		return nil, err
	}

	defer body.Close()
	var workspace Workspace
	err = json.NewDecoder(body).Decode(&workspace)
	if err != nil {
		return nil, err
	}

	return &workspace, nil
}

func (p *Postman) DeleteWorkspace(id string) error {
	_, err := p.performRequest("DELETE", fmt.Sprintf("%sworkspaces/%s", url, id), nil)
	if err != nil {
		return err
	}

	return nil
}

/*************************************************************************************************/
/***************************************** COLLECTION ********************************************/
/*************************************************************************************************/
func (p *Postman) RetrieveCollections() ([]CollectionInfo, error) {
	body, err := p.performRequest("GET", fmt.Sprintf("%scollections", url), nil)
	if err != nil {
		return nil, err
	}

	defer body.Close()
	var resp CollectionListResponse
	err = json.NewDecoder(body).Decode(&resp)
	if err != nil {
		return nil, err
	}

	return resp.Collections, nil
}

func (p *Postman) RetrieveCollection(uid string) (*Collection, error) {
	body, err := p.performRequest("GET", fmt.Sprintf("%scollections/%s", url, uid), nil)
	if err != nil {
		return nil, err
	}

	defer body.Close()
	var resp CollectionResponse
	err = json.NewDecoder(body).Decode(&resp)
	if err != nil {
		return nil, err
	}

	return &resp.Collection, nil
}

func (p *Postman) CreateCollections(c Collection, wsUid string) (*Collection, error) {
	reqBody, err := json.Marshal(CollectionResponse{Collection: c})
	if err != nil {
		return nil, err
	}
	body, err := p.performRequest("POST", fmt.Sprintf("%scollections?workspace=%s", url, wsUid), reqBody)
	if err != nil {
		return nil, err
	}

	defer body.Close()
	var collection Collection
	err = json.NewDecoder(body).Decode(&collection)
	if err != nil {
		return nil, err
	}

	return &collection, nil
}

func (p *Postman) UpdateCollection(uid string, c Collection) (*Collection, error) {
	reqBody, err := json.Marshal(CollectionResponse{Collection: c})
	if err != nil {
		return nil, err
	}
	body, err := p.performRequest("PUT", fmt.Sprintf("%scollections/%s", url, uid), reqBody)
	if err != nil {
		return nil, err
	}

	defer body.Close()
	var collection Collection
	err = json.NewDecoder(body).Decode(&collection)
	if err != nil {
		return nil, err
	}

	return &collection, nil
}

func (p *Postman) DeleteCollection(id string) error {
	_, err := p.performRequest("DELETE", fmt.Sprintf("%scollections/%s", url, id), nil)
	if err != nil {
		return err
	}

	return nil
}

// /*************************************************************************************************/
// /**************************************** ENVIRONMENT ********************************************/
// /*************************************************************************************************/
func (p *Postman) RetrieveEnvironments() ([]EnvironmentInfo, error) {
	body, err := p.performRequest("GET", fmt.Sprintf("%senvironments", url), nil)
	if err != nil {
		return nil, err
	}

	defer body.Close()
	var resp EnvironmentListResponse
	err = json.NewDecoder(body).Decode(&resp)
	if err != nil {
		return nil, err
	}

	return resp.Environments, nil
}

func (p *Postman) RetrieveEnvironment(uid string) (*Environment, error) {
	body, err := p.performRequest("GET", fmt.Sprintf("%senvironments/%s", url, uid), nil)
	if err != nil {
		return nil, err
	}

	defer body.Close()
	var resp EnvironmentResponse
	err = json.NewDecoder(body).Decode(&resp)
	if err != nil {
		return nil, err
	}

	return &resp.Environment, nil
}

func (p *Postman) CreateEnvironments(e Environment, wsUid string) (*Environment, error) {
	reqBody, err := json.Marshal(EnvironmentResponse{Environment: e})
	if err != nil {
		return nil, err
	}
	body, err := p.performRequest("POST", fmt.Sprintf("%senvironments?workspace=%s", url, wsUid), reqBody)
	if err != nil {
		return nil, err
	}

	defer body.Close()
	var environment Environment
	err = json.NewDecoder(body).Decode(&environment)
	if err != nil {
		return nil, err
	}

	return &environment, nil
}

func (p *Postman) UpdateEnvironment(uid string, e Environment) (*Environment, error) {
	reqBody, err := json.Marshal(EnvironmentResponse{Environment: e})
	if err != nil {
		return nil, err
	}
	body, err := p.performRequest("PUT", fmt.Sprintf("%senvironments/%s", url, uid), reqBody)
	if err != nil {
		return nil, err
	}

	defer body.Close()
	var environment Environment
	err = json.NewDecoder(body).Decode(&environment)
	if err != nil {
		return nil, err
	}

	return &environment, nil
}

func (p *Postman) DeleteEnvironment(uid string) error {
	_, err := p.performRequest("DELETE", fmt.Sprintf("%senvironments/%s", url, uid), nil)
	if err != nil {
		return err
	}

	return nil
}

// Method used to perform all fetch requests
func (p *Postman) performRequest(method, url string, reqBody []byte) (io.ReadCloser, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Add(apiKeyHeader, p.ApiKey)
	req.Header.Add(contentType, contentTypeJSON)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// HTTPClientWithFallBack to be used in all fetch operations.
func httpClientWithFallBack(h *http.Client) *http.Client {
	if h != nil {
		return h
	}
	return http.DefaultClient
}
