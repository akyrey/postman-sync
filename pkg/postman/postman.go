package postman

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"log/slog"
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

	var resp WorkspaceListResponse
	err = p.getJsonResponse(body, &resp)
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

	var resp WorkspaceResponse
	err = p.getJsonResponse(body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.Workspace, nil
}

func (p *Postman) CreateWorkspace(ws CreateWorkspaceParam) (*Workspace, error) {
	reqBody, err := json.Marshal(struct {
		Workspace CreateWorkspaceParam `json:"workspace"`
	}{Workspace: ws})
	if err != nil {
		slog.Error("Error marshaling workspace", err)
		return nil, err
	}
	body, err := p.performRequest("POST", fmt.Sprintf("%sworkspaces", url), reqBody)
	if err != nil {
		return nil, err
	}

	var resp WorkspaceResponse
	err = p.getJsonResponse(body, &resp)
	if err != nil {
		slog.Error("Error decoding response to workspace", err)
		return nil, err
	}

	return &resp.Workspace, nil
}

func (p *Postman) UpdateWorkspace(uid string, ws UpdateWorkspaceParam) (*Workspace, error) {
	reqBody, err := json.Marshal(struct {
		Workspace UpdateWorkspaceParam `json:"workspace"`
	}{Workspace: ws})
	if err != nil {
		return nil, err
	}
	body, err := p.performRequest("PUT", fmt.Sprintf("%sworkspaces/%s", url, uid), reqBody)
	if err != nil {
		return nil, err
	}

	var resp WorkspaceResponse
	err = p.getJsonResponse(body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.Workspace, nil
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

	var resp CollectionListResponse
	err = p.getJsonResponse(body, &resp)
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

	var resp CollectionResponse
	err = p.getJsonResponse(body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.Collection, nil
}

func (p *Postman) CreateCollection(c CreateCollectionParam, wsUid string) (*Collection, error) {
	reqBody, err := json.Marshal(struct {
		Collection CreateCollectionParam `json:"collection"`
	}{Collection: c})
	if err != nil {
		return nil, err
	}
	body, err := p.performRequest("POST", fmt.Sprintf("%scollections?workspace=%s", url, wsUid), reqBody)
	if err != nil {
		return nil, err
	}

	var resp CollectionResponse
	err = p.getJsonResponse(body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.Collection, nil
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

	var resp CollectionResponse
	err = p.getJsonResponse(body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.Collection, nil
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

	var resp EnvironmentListResponse
	err = p.getJsonResponse(body, &resp)
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

	var resp EnvironmentResponse
	err = p.getJsonResponse(body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.Environment, nil
}

func (p *Postman) CreateEnvironment(e CreateEnvironmentParam, wsUid string) (*Environment, error) {
	reqBody, err := json.Marshal(struct {
		Environment CreateEnvironmentParam `json:"environment"`
	}{Environment: e})
	if err != nil {
		return nil, err
	}
	body, err := p.performRequest("POST", fmt.Sprintf("%senvironments?workspace=%s", url, wsUid), reqBody)
	if err != nil {
		return nil, err
	}

	var resp EnvironmentResponse
	err = p.getJsonResponse(body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.Environment, nil
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

	var resp EnvironmentResponse
	err = p.getJsonResponse(body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.Environment, nil
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
	// fmt.Printf("Performing request %s %s %s\n", method, url, string(reqBody))
	slog.Debug("Performing request", method, url, string(reqBody))
	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		slog.Error("Failed creating request", err)
		return nil, err
	}

	req.Header.Add(apiKeyHeader, p.ApiKey)
	req.Header.Add(contentType, contentTypeJSON)

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Failed performing request", err)
		return nil, err
	}
	// body, err := io.ReadAll(resp.Body)
	// fmt.Printf("Received response %v\n", string(body))
	//    resp.Body = io.NopCloser(bytes.NewReader(body))

	return resp.Body, nil
}

func (p *Postman) getJsonResponse(body io.ReadCloser, target interface{}) error {
	defer body.Close()
	return json.NewDecoder(body).Decode(target)
}

// HTTPClientWithFallBack to be used in all fetch operations.
func httpClientWithFallBack(h *http.Client) *http.Client {
	if h != nil {
		return h
	}
	return http.DefaultClient
}
