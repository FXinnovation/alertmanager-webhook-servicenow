package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/prometheus/common/log"
)

const (
	serviceNowBaseURL = "https://%s.service-now.com"
	tableAPI          = "%s/api/now/v2/table/%s"
)

// Incident is a model of the ServiceNow incident table
type Incident struct {
	AssignmentGroup  string      `json:"assignment_group"`
	ContactType      string      `json:"contact_type"`
	CallerID         string      `json:"caller_id"`
	Comments         string      `json:"comments"`
	Description      string      `json:"description"`
	Impact           json.Number `json:"impact"`
	Priority         string      `json:"priority"`
	ShortDescription string      `json:"short_description"`
	State            json.Number `json:"state"`
	Urgency          json.Number `json:"urgency"`
}

// ServiceNow interface
type ServiceNow interface {
	CreateIncident(incident Incident) (string, error)
	GetIncident(params map[string]string) (string, error)
}

// ServiceNowClient is the interface to a ServiceNow instance
type ServiceNowClient struct {
	baseURL    string
	authHeader string
	client     *http.Client
}

// NewServiceNowClient will create a new ServiceNow client
func NewServiceNowClient(instanceName string, userName string, password string) (*ServiceNowClient, error) {
	if instanceName == "" {
		return nil, errors.New("Missing instanceName")
	}

	if userName == "" {
		return nil, errors.New("Missing userName")
	}

	if password == "" {
		return nil, errors.New("Missing password")
	}

	return &ServiceNowClient{
		baseURL:    fmt.Sprintf(serviceNowBaseURL, instanceName),
		authHeader: fmt.Sprintf("Basic %s", base64.URLEncoding.EncodeToString([]byte(userName+":"+password))),
		client:     http.DefaultClient,
	}, nil
}

// Create a table item in ServiceNow from a post body
func (snClient *ServiceNowClient) create(table string, body []byte) (string, error) {
	log.Infof("Create a ServiceNow %s", table)
	url := fmt.Sprintf(tableAPI, snClient.baseURL, table)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Errorf("Error creating the request. %s", err)
		return "", err
	}

	return snClient.doRequest(req)
}

// get a table item from ServiceNow using a map of arguments
func (snClient *ServiceNowClient) get(table string, params map[string]string) (string, error) {
	log.Infof("Get a ServiceNow %s", table)
	url := fmt.Sprintf(tableAPI, snClient.baseURL, table)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorf("Error creating the request. %s", err)
		return "", err
	}

	q := req.URL.Query()
	for key, val := range params {
		q.Add(key, val)
	}
	req.URL.RawQuery = q.Encode()

	return snClient.doRequest(req)
}

// doRequest will do the given ServiceNow request and return response as string
func (snClient *ServiceNowClient) doRequest(req *http.Request) (string, error) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", snClient.authHeader)
	resp, err := snClient.client.Do(req)
	if err != nil {
		log.Errorf("Error sending the request. %s", err)
		return "", err
	}
	if resp.StatusCode >= 400 {
		errorMsg := fmt.Sprintf("ServiceNow returned the HTTP error code: %v", resp.StatusCode)
		log.Error(errorMsg)
		return "", errors.New(errorMsg)
	}

	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error reading the body. %s", err)
		return "", err
	}

	return string(responseBody), nil
}

// CreateIncident will create an incident in ServiceNow from an Incident
func (snClient *ServiceNowClient) CreateIncident(incident Incident) (string, error) {
	postBody, err := json.Marshal(incident)
	if err != nil {
		log.Errorf("Error while marshalling the incident. %s", err)
		return "", err
	}

	response, err := snClient.create("incident", postBody)
	if err != nil {
		log.Errorf("Error while creating the incident. %s", err)
		return "", err
	}

	return response, nil
}

// GetIncident will retrieve an incident from ServiceNow
func (snClient *ServiceNowClient) GetIncident(params map[string]string) (string, error) {
	response, err := snClient.get("incident", params)

	if err != nil {
		log.Errorf("Error while getting the incident. %s", err)
		return "", err
	}

	return response, nil
}
