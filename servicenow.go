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
type Incident map[string]interface{}

// GetSysID returns the sys_id of the incident
func (i Incident) GetSysID() string {
	return i["sys_id"].(string)
}

// GetNumber returns the number of the incident
func (i Incident) GetNumber() string {
	return i["number"].(string)
}

// GetState returns the state of the incident
func (i Incident) GetState() json.Number {
	return json.Number(i["state"].(string))
}

// IncidentResponse is a model of an API response contaning one incident
type IncidentResponse map[string]interface{}

// GetResult returns the incident from the IncidentResponse
func (ir IncidentResponse) GetResult() Incident {
	var incident Incident = ir["result"].(map[string]interface{})
	return incident
}

// IncidentsResponse is a model of an API response contaning multiple incidents
type IncidentsResponse map[string]interface{}

// GetResults returns the incidents from the IncidentsResponse
func (ir IncidentsResponse) GetResults() []Incident {
	results := ir["result"].([]interface{})
	incidents := make([]Incident, len(results))
	for i, result := range results {
		incidents[i] = result.(map[string]interface{})
	}
	return incidents
}

// ServiceNow interface
type ServiceNow interface {
	CreateIncident(incidentParam Incident) (Incident, error)
	GetIncidents(params map[string]string) ([]Incident, error)
	UpdateIncident(incidentParam Incident, sysID string) (Incident, error)
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
func (snClient *ServiceNowClient) create(table string, body []byte) ([]byte, error) {
	url := fmt.Sprintf(tableAPI, snClient.baseURL, table)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Errorf("Error creating the request. %s", err)
		return nil, err
	}

	return snClient.doRequest(req)
}

// get a table item from ServiceNow using a map of arguments
func (snClient *ServiceNowClient) get(table string, params map[string]string) ([]byte, error) {
	url := fmt.Sprintf(tableAPI, snClient.baseURL, table)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorf("Error creating the request. %s", err)
		return nil, err
	}

	q := req.URL.Query()
	for key, val := range params {
		q.Add(key, val)
	}
	req.URL.RawQuery = q.Encode()

	return snClient.doRequest(req)
}

// update a table item in ServiceNow from a post body and a sys_id
func (snClient *ServiceNowClient) update(table string, body []byte, sysID string) ([]byte, error) {
	url := fmt.Sprintf(tableAPI+"/%s", snClient.baseURL, table, sysID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		log.Errorf("Error creating the request. %s", err)
		return nil, err
	}

	return snClient.doRequest(req)
}

// doRequest will do the given ServiceNow request and return response as byte array
func (snClient *ServiceNowClient) doRequest(req *http.Request) ([]byte, error) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", snClient.authHeader)
	resp, err := snClient.client.Do(req)
	if err != nil {
		log.Errorf("Error sending the request. %s", err)
		return nil, err
	}
	if resp.StatusCode >= 400 {
		errorMsg := fmt.Sprintf("ServiceNow returned the HTTP error code: %v", resp.StatusCode)
		log.Error(errorMsg)
		return nil, errors.New(errorMsg)
	}

	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error reading the body. %s", err)
		return nil, err
	}

	return responseBody, nil
}

// CreateIncident will create an incident in ServiceNow from a given Incident, and return the created incident
func (snClient *ServiceNowClient) CreateIncident(incidentParam Incident) (Incident, error) {
	log.Info("Create a ServiceNow incident")

	postBody, err := json.Marshal(incidentParam)
	if err != nil {
		log.Errorf("Error while marshalling the incident. %s", err)
		return nil, err
	}

	response, err := snClient.create("incident", postBody)
	if err != nil {
		log.Errorf("Error while creating the incident. %s", err)
		return nil, err
	}

	incidentResponse := IncidentResponse{}
	err = json.Unmarshal(response, &incidentResponse)
	if err != nil {
		log.Errorf("Error while unmarshalling the incident. %s", err)
		return nil, err
	}

	createdIncident := incidentResponse.GetResult()
	log.Infof("Incident %s created", createdIncident.GetNumber())

	return createdIncident, nil
}

// GetIncidents will retrieve an incident from ServiceNow
func (snClient *ServiceNowClient) GetIncidents(params map[string]string) ([]Incident, error) {
	log.Infof("Get ServiceNow incidents with params: %v", params)
	response, err := snClient.get("incident", params)

	if err != nil {
		log.Errorf("Error while getting the incident. %s", err)
		return nil, err
	}

	incidentsResponse := IncidentsResponse{}
	err = json.Unmarshal(response, &incidentsResponse)
	if err != nil {
		log.Errorf("Error while unmarshalling the incident. %s", err)
		return nil, err
	}

	return incidentsResponse.GetResults(), nil
}

// UpdateIncident will update an incident in ServiceNow from a given Incident, and return the updated incident
func (snClient *ServiceNowClient) UpdateIncident(incidentParam Incident, sysID string) (Incident, error) {
	log.Infof("Update %v field(s) of ServiceNow incident with id : %s", len(incidentParam), sysID)

	postBody, err := json.Marshal(incidentParam)
	if err != nil {
		log.Errorf("Error while marshalling the incident. %s", err)
		return nil, err
	}

	response, err := snClient.update("incident", postBody, sysID)
	if err != nil {
		log.Errorf("Error while updating the incident. %s", err)
		return nil, err
	}

	incidentResponse := IncidentResponse{}
	err = json.Unmarshal(response, &incidentResponse)
	if err != nil {
		log.Errorf("Error while unmarshalling the incident. %s", err)
		return nil, err
	}

	updatedIncident := incidentResponse.GetResult()
	log.Infof("Incident %s updated", updatedIncident.GetNumber())

	return updatedIncident, nil
}
