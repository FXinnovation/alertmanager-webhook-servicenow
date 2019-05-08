package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/prometheus/common/log"
)

const (
	serviceNowBaseURL = "https://%s.service-now.com"
	tableAPI          = "%s/api/now/table/%s"
)

// Incident is a model of the ServiceNow incident table
type Incident struct {
	AssignmentGroup  string      `json:"assignment_group"`
	ContactType      string      `json:"contact_type"`
	CallerID         string      `json:"caller_id"`
	Description      string      `json:"description"`
	Impact           json.Number `json:"impact"`
	ShortDescription string      `json:"short_description"`
	State            json.Number `json:"state"`
	Urgency          json.Number `json:"urgency"`
}

// ServiceNowClient is the interface to a ServiceNow instance
type ServiceNowClient struct {
	instanceName string
	baseURL      string
	authHeader   string

	client *http.Client
}

// NewServiceNowClient will create a new ServiceNow client
func NewServiceNowClient(instanceName string, userName string, password string) *ServiceNowClient {
	return &ServiceNowClient{
		instanceName: instanceName,
		baseURL:      fmt.Sprintf(serviceNowBaseURL, instanceName),
		authHeader:   fmt.Sprintf("Basic %s", base64.URLEncoding.EncodeToString([]byte(userName+":"+password))),
		client:       http.DefaultClient,
	}
}

//Create a table item in ServiceNow from a post body
func (snClient *ServiceNowClient) create(table string, body []byte) (string, error) {
	log.Infof("Creating a ServiceNow %s", table)
	url := fmt.Sprintf("%s/api/now/table/%s", snClient.baseURL, table)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Errorf("Error creating the request. %s", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", snClient.authHeader)
	resp, err := snClient.client.Do(req)
	if err != nil {
		log.Errorf("Error sending the request. %s", err)
		return "", err
	}
	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error reading the body. %s", err)
		return "", err
	}

	fmt.Printf("responseBody: %s", string(responseBody))
	return string(responseBody), nil
}
