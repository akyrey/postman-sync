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
func (p *Postman) RetrieveWorkspaces() {
	body, err := p.performRequest("GET", fmt.Sprintf("%sworkspaces/", url), nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}

func (p *Postman) RetrieveWorkspace(id string) {
	body, err := p.performRequest("GET", fmt.Sprintf("%sworkspaces/%s", url, id), nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}

func (p *Postman) CreateWorkspace(workspace Workspace) {
	reqBody, err := json.Marshal(struct {
		Workspace Workspace `json:"workspace"`
	}{Workspace: workspace})
	if err != nil {
		panic(err)
	}
	body, err := p.performRequest("POST", fmt.Sprintf("%sworkspaces", url), reqBody)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}

func (p *Postman) UpdateWorkspace(id string, workspace Workspace) {
	reqBody, err := json.Marshal(struct {
		Workspace Workspace `json:"workspace"`
	}{Workspace: workspace})
	if err != nil {
		panic(err)
	}
	body, err := p.performRequest("PUT", fmt.Sprintf("%sworkspaces/%s", url, id), reqBody)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}

func (p *Postman) DeleteWorkspace(id string) {
	body, err := p.performRequest("DELETE", fmt.Sprintf("%sworkspaces/%s", url, id), nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}

/*************************************************************************************************/
/***************************************** COLLECTION ********************************************/
/*************************************************************************************************/
func (p *Postman) RetrieveCollections() {
	body, err := p.performRequest("GET", fmt.Sprintf("%scollections", url), nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}

func (p *Postman) RetrieveCollection(id string) {
	body, err := p.performRequest("GET", fmt.Sprintf("%scollections/%s", url, id), nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}

func (p *Postman) CreateCollections(collection Collection) {
	reqBody, err := json.Marshal(collection)
	if err != nil {
		panic(err)
	}
	body, err := p.performRequest("POST", fmt.Sprintf("%scollections", url), reqBody)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}

func (p *Postman) UpdateCollection(id string, collection Collection) {
	reqBody, err := json.Marshal(collection)
	if err != nil {
		panic(err)
	}
	body, err := p.performRequest("PUT", fmt.Sprintf("%scollections/%s", url, id), reqBody)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}

func (p *Postman) DeleteCollection(id string) {
	body, err := p.performRequest("DELETE", fmt.Sprintf("%scollections/%s", url, id), nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}

/*************************************************************************************************/
/**************************************** ENVIRONMENT ********************************************/
/*************************************************************************************************/
func (p *Postman) RetrieveEnvironments() {
	body, err := p.performRequest("GET", fmt.Sprintf("%senvironments", url), nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}

func (p *Postman) RetrieveEnvironment(id string) {
	body, err := p.performRequest("GET", fmt.Sprintf("%senvironments/%s", url, id), nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}

func (p *Postman) CreateEnvironments(environment Environment) {
	reqBody, err := json.Marshal(environment)
	if err != nil {
		panic(err)
	}
	body, err := p.performRequest("POST", fmt.Sprintf("%senvironments", url), reqBody)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}

func (p *Postman) UpdateEnvironment(id string, environment Environment) {
	reqBody, err := json.Marshal(environment)
	if err != nil {
		panic(err)
	}
	body, err := p.performRequest("PUT", fmt.Sprintf("%senvironments/%s", url, id), reqBody)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}

func (p *Postman) DeleteEnvironment(id string) {
	body, err := p.performRequest("DELETE", fmt.Sprintf("%senvironments/%s", url, id), nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(body))
}

// Method used to perform all fetch requests
func (p *Postman) performRequest(method, url string, reqBody []byte) ([]byte, error) {
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

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// HTTPClientWithFallBack to be used in all fetch operations.
func httpClientWithFallBack(h *http.Client) *http.Client {
	if h != nil {
		return h
	}
	return http.DefaultClient
}
