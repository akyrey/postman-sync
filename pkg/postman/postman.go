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
func (p *Postman) RetrieveWorkspaces() ([]Workspace, error) {
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

	return resp.Workspaces, err
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

	return &resp.Workspace, err
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

	return &workspace, err
}

func (p *Postman) UpdateWorkspace(id string, ws Workspace) (*Workspace, error) {
	reqBody, err := json.Marshal(WorkspaceResponse{Workspace: ws})
	if err != nil {
		return nil, err
	}
	body, err := p.performRequest("PUT", fmt.Sprintf("%sworkspaces/%s", url, id), reqBody)
	if err != nil {
		return nil, err
	}

	defer body.Close()
	var workspace Workspace
	err = json.NewDecoder(body).Decode(&workspace)
	if err != nil {
		return nil, err
	}

	return &workspace, err
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
// func (p *Postman) RetrieveCollections() {
// 	body, err := p.performRequest("GET", fmt.Sprintf("%scollections", url), nil)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	fmt.Println(string(body))
// }
//
// func (p *Postman) RetrieveCollection(id string) {
// 	body, err := p.performRequest("GET", fmt.Sprintf("%scollections/%s", url, id), nil)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	fmt.Println(string(body))
// }
//
// func (p *Postman) CreateCollections(collection Collection) {
// 	reqBody, err := json.Marshal(collection)
// 	if err != nil {
// 		panic(err)
// 	}
// 	body, err := p.performRequest("POST", fmt.Sprintf("%scollections", url), reqBody)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	fmt.Println(string(body))
// }
//
// func (p *Postman) UpdateCollection(id string, collection Collection) {
// 	reqBody, err := json.Marshal(collection)
// 	if err != nil {
// 		panic(err)
// 	}
// 	body, err := p.performRequest("PUT", fmt.Sprintf("%scollections/%s", url, id), reqBody)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	fmt.Println(string(body))
// }
//
// func (p *Postman) DeleteCollection(id string) {
// 	body, err := p.performRequest("DELETE", fmt.Sprintf("%scollections/%s", url, id), nil)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	fmt.Println(string(body))
// }
//
// /*************************************************************************************************/
// /**************************************** ENVIRONMENT ********************************************/
// /*************************************************************************************************/
// func (p *Postman) RetrieveEnvironments() {
// 	body, err := p.performRequest("GET", fmt.Sprintf("%senvironments", url), nil)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	fmt.Println(string(body))
// }
//
// func (p *Postman) RetrieveEnvironment(id string) {
// 	body, err := p.performRequest("GET", fmt.Sprintf("%senvironments/%s", url, id), nil)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	fmt.Println(string(body))
// }
//
// func (p *Postman) CreateEnvironments(environment Environment) {
// 	reqBody, err := json.Marshal(environment)
// 	if err != nil {
// 		panic(err)
// 	}
// 	body, err := p.performRequest("POST", fmt.Sprintf("%senvironments", url), reqBody)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	fmt.Println(string(body))
// }
//
// func (p *Postman) UpdateEnvironment(id string, environment Environment) {
// 	reqBody, err := json.Marshal(environment)
// 	if err != nil {
// 		panic(err)
// 	}
// 	body, err := p.performRequest("PUT", fmt.Sprintf("%senvironments/%s", url, id), reqBody)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	fmt.Println(string(body))
// }
//
// func (p *Postman) DeleteEnvironment(id string) {
// 	body, err := p.performRequest("DELETE", fmt.Sprintf("%senvironments/%s", url, id), nil)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	fmt.Println(string(body))
// }

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
