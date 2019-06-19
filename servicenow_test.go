package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

var basicIncidentParam = Incident{
	"assignment_group":  "42",
	"caller_id":         "Prometheus",
	"description":       "This is the description",
	"short_description": "This is the short description",
	"impact":            "4",
	"state":             "0",
	"urgency":           "3",
}

var wrongIncidentParam = Incident{
	"impact": "4xxx",
}

func TestNewServiceNowClient_OK(t *testing.T) {
	snClient, err := NewServiceNowClient("instanceName", "userName", "password")

	if err != nil {
		t.Errorf("Error occured %s", err)
	}

	expectedBaseURL := "https://instanceName.service-now.com"
	if snClient.baseURL != expectedBaseURL {
		t.Errorf("Unexpected baseURL; got: %v, want: %v", snClient.baseURL, expectedBaseURL)
	}

	expectedAuthHeader := "Basic dXNlck5hbWU6cGFzc3dvcmQ="
	if snClient.authHeader != expectedAuthHeader {
		t.Errorf("Unexpected authHeader; got: %v, want: %v", snClient.authHeader, expectedAuthHeader)
	}

	if reflect.TypeOf(&http.Client{}) != reflect.TypeOf(snClient.client) {
		t.Errorf("Unexpected client type; got: %v, want: %v", reflect.TypeOf(snClient.client), reflect.TypeOf(&http.Client{}))
	}
}

func TestNewServiceNowClient_MissingInstanceName(t *testing.T) {
	_, err := NewServiceNowClient("", "userName", "password")

	if err == nil {
		t.Errorf("Expected an error, got none")
	}
}

func TestNewServiceNowClient_MissingUserName(t *testing.T) {
	_, err := NewServiceNowClient("instancename", "", "password")

	if err == nil {
		t.Errorf("Expected an error, got none")
	}
}

func TestNewServiceNowClient_MissingPassword(t *testing.T) {
	_, err := NewServiceNowClient("instancename", "userName", "")

	if err == nil {
		t.Errorf("Expected an error, got none")
	}
}

func TestCreateIncident_OK(t *testing.T) {
	// Load a simple example of a response coming from ServiceNow
	incidentTest, err := ioutil.ReadFile("test/incident_response.json")
	if err != nil {
		t.Fatal(err)
	}
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, string(incidentTest))
	}

	ts := httptest.NewServer(http.HandlerFunc(testHandler))
	defer ts.Close()

	snClient, err := NewServiceNowClient("instancename", "username", "password")
	snClient.baseURL = ts.URL

	if err != nil {
		t.Errorf("Error occured on NewServiceNowClient: %s", err)
	}

	incident, err := snClient.CreateIncident(basicIncidentParam)

	if err != nil {
		t.Errorf("Error occured on CreateIncident: %s", err)
	}

	expectedIncidentResponse := IncidentResponse{}
	_ = json.Unmarshal(incidentTest, &expectedIncidentResponse)

	if !reflect.DeepEqual(incident, expectedIncidentResponse.GetResult()) {
		t.Errorf("Unexpected response; got: %v, want: %v", incident, expectedIncidentResponse.GetResult())
	}
}

func TestCreateIncident_OK_No_AG(t *testing.T) {
	// Load a simple example of a response coming from ServiceNow
	incidentTest, err := ioutil.ReadFile("test/incident_response_no_ag.json")
	if err != nil {
		t.Fatal(err)
	}
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, string(incidentTest))
	}

	ts := httptest.NewServer(http.HandlerFunc(testHandler))
	defer ts.Close()

	snClient, err := NewServiceNowClient("instancename", "username", "password")
	snClient.baseURL = ts.URL

	if err != nil {
		t.Errorf("Error occured on NewServiceNowClient: %s", err)
	}

	incident, err := snClient.CreateIncident(basicIncidentParam)

	if err != nil {
		t.Errorf("Error occured on CreateIncident: %s", err)
	}

	expectedIncidentResponse := IncidentResponse{}
	_ = json.Unmarshal(incidentTest, &expectedIncidentResponse)

	if !reflect.DeepEqual(incident, expectedIncidentResponse.GetResult()) {
		t.Errorf("Unexpected response; got: %v, want: %v", incident, expectedIncidentResponse.GetResult())
	}
}

func TestCreateIncident_IncidentMarshallError(t *testing.T) {
	testHandler := func(w http.ResponseWriter, r *http.Request) {}
	ts := httptest.NewServer(http.HandlerFunc(testHandler))
	defer ts.Close()

	snClient, err := NewServiceNowClient("instancename", "username", "password")
	snClient.baseURL = ts.URL

	if err != nil {
		t.Errorf("Error occured on NewServiceNowClient: %s", err)
	}

	// Cause an error by using invalid incident
	_, err = snClient.CreateIncident(wrongIncidentParam)

	if err == nil {
		t.Errorf("Expected an error, got none")
	}
}

func TestCreateIncident_CreateRequestError(t *testing.T) {
	snClient, err := NewServiceNowClient("instancename", "username", "password")
	// Cause an error by using an invalid URL
	snClient.baseURL = "very bad url"

	if err != nil {
		t.Errorf("Error occured on NewServiceNowClient: %s", err)
	}

	_, err = snClient.CreateIncident(basicIncidentParam)

	if err == nil {
		t.Errorf("Expected an error, got none")
	}
}

func TestCreateIncident_DoRequestError(t *testing.T) {
	testHandler := func(w http.ResponseWriter, r *http.Request) {}
	ts := httptest.NewServer(http.HandlerFunc(testHandler))

	snClient, err := NewServiceNowClient("instancename", "username", "password")
	snClient.baseURL = ts.URL

	if err != nil {
		t.Errorf("Error occured on NewServiceNowClient: %s", err)
	}

	// Cause an error by closing the server
	ts.Close()
	_, err = snClient.CreateIncident(basicIncidentParam)

	if err == nil {
		t.Errorf("Expected an error, got none")
	}
}

func TestCreateIncident_InternalServerError(t *testing.T) {
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		// Cause an error by simulating HTTP code 500
		w.WriteHeader(http.StatusInternalServerError)
	}
	ts := httptest.NewServer(http.HandlerFunc(testHandler))
	defer ts.Close()

	snClient, err := NewServiceNowClient("instancename", "username", "password")
	snClient.baseURL = ts.URL

	if err != nil {
		t.Errorf("Error occured on NewServiceNowClient: %s", err)
	}

	_, err = snClient.CreateIncident(basicIncidentParam)

	if err == nil {
		t.Errorf("Expected an error, got none")
	}
}

func TestGetIncidents_OK(t *testing.T) {
	// Load a simple example of a response coming from ServiceNow
	incidentsTest, err := ioutil.ReadFile("test/get_incidents_response.json")
	if err != nil {
		t.Fatal(err)
	}
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, string(incidentsTest))
	}
	ts := httptest.NewServer(http.HandlerFunc(testHandler))
	defer ts.Close()

	snClient, err := NewServiceNowClient("instancename", "username", "password")
	snClient.baseURL = ts.URL
	if err != nil {
		t.Errorf("Error occured on NewServiceNowClient: %s", err)
	}

	incidents, err := snClient.GetIncidents(nil)
	if err != nil {
		t.Errorf("Error occured on CreateIncident: %s", err)
	}

	expectedIncidentsResponse := IncidentsResponse{}
	_ = json.Unmarshal(incidentsTest, &expectedIncidentsResponse)

	if !reflect.DeepEqual(incidents, expectedIncidentsResponse.GetResults()) {
		t.Errorf("Unexpected incident; got: %v, want: %v", incidents, expectedIncidentsResponse.GetResults())
	}
}

func TestGetIncidents_CreateRequestError(t *testing.T) {
	snClient, err := NewServiceNowClient("instancename", "username", "password")
	// Cause an error by using an invalid URL
	snClient.baseURL = "very bad url"

	if err != nil {
		t.Errorf("Error occured on NewServiceNowClient: %s", err)
	}

	_, err = snClient.GetIncidents(nil)

	if err == nil {
		t.Errorf("Expected an error, got none")
	}
}

func TestUpdateIncident_OK(t *testing.T) {
	// Load a simple example of a response coming from ServiceNow
	incidentTest, err := ioutil.ReadFile("test/incident_response.json")
	if err != nil {
		t.Fatal(err)
	}
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, string(incidentTest))
	}

	ts := httptest.NewServer(http.HandlerFunc(testHandler))
	defer ts.Close()

	snClient, err := NewServiceNowClient("instancename", "username", "password")
	snClient.baseURL = ts.URL

	if err != nil {
		t.Errorf("Error occured on NewServiceNowClient: %s", err)
	}

	incident, err := snClient.UpdateIncident(basicIncidentParam, "my_sys_id")

	if err != nil {
		t.Errorf("Error occured on UpdateIncident: %s", err)
	}

	expectedIncidentResponse := IncidentResponse{}
	_ = json.Unmarshal(incidentTest, &expectedIncidentResponse)

	if !reflect.DeepEqual(incident, expectedIncidentResponse.GetResult()) {
		t.Errorf("Unexpected response; got: %v, want: %v", incident, expectedIncidentResponse.GetResult())
	}
}

func TestUpdateIncident_CreateRequestError(t *testing.T) {
	snClient, err := NewServiceNowClient("instancename", "username", "password")
	// Cause an error by using an invalid URL
	snClient.baseURL = "very bad url"

	if err != nil {
		t.Errorf("Error occured on NewServiceNowClient: %s", err)
	}

	_, err = snClient.UpdateIncident(basicIncidentParam, "my_sys_id")

	if err == nil {
		t.Errorf("Expected an error, got none")
	}
}
