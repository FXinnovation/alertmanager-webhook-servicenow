package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/common/log"
)

const (
	serviceNowBaseURL = "https://%s.service-now.com"
	tableAPI          = "%s/api/now/v2/table/%s"
)

// Config - ServiceNow webhook configuration
type Config struct {
	ServiceNow ServiceNowConfig `yaml:"service_now"`
}

// ServiceNowConfig - ServiceNow instance configuration
type ServiceNowConfig struct {
	InstanceName string `yaml:"instance_name"`
	UserName     string `yaml:"user_name"`
	Password     string `yaml:"password"`
}

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
	log.Infof("Creating a ServiceNow %s", table)
	url := fmt.Sprintf(tableAPI, snClient.baseURL, table)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Errorf("Error creating the request. %s", err)
		return "", err
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

// ManageIncidents implement logic to manage incidents based on AlertManager Data
func (snClient *ServiceNowClient) ManageIncidents(data template.Data, config Config) error {

	log.Infof("Alerts: Status=%s, GroupLabels=%v, CommonLabels=%v", data.Status, data.GroupLabels, data.CommonLabels)

	for _, alert := range data.Alerts {
		incident := snClient.alertToIncident(alert)
		_, err := snClient.CreateIncident(incident)

		if err != nil {
			log.Errorf("Error while creating incident: %v", err)
			return err
		}
	}

	return nil
}

func (snClient *ServiceNowClient) alertToIncident(alert template.Alert) Incident {
	incident := Incident{
		AssignmentGroup:  alert.Labels["assignment_group"],
		ContactType:      "Monitoring System",
		CallerID:         "Prometheus",
		Description:      alert.Annotations["description"],
		Impact:           "4",
		ShortDescription: alert.Annotations["summary"],
		State:            json.Number("60"),
		Urgency:          "3",
	}
	return incident
}
