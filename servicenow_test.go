package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

var basicIncident = Incident{
	CallerID:         "Prometheus",
	Description:      "This is the description",
	ShortDescription: "This is the short description",
	Impact:           "4",
	Urgency:          "3",
}

var wrongIncident = Incident{
	Impact: "4xxx",
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
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"sys_id":"424242","number":"INC42"}`)
	}
	ts := httptest.NewServer(http.HandlerFunc(testHandler))
	defer ts.Close()

	snClient, err := NewServiceNowClient("instancename", "username", "password")
	snClient.baseURL = ts.URL

	if err != nil {
		t.Errorf("Error occured on NewServiceNowClient: %s", err)
	}

	response, err := snClient.CreateIncident(basicIncident)

	if err != nil {
		t.Errorf("Error occured on CreateIncident: %s", err)
	}

	expectedResponse := `{"sys_id":"424242","number":"INC42"}`
	if response != expectedResponse {
		t.Errorf("Unexpected response; got: %v, want: %v", response, expectedResponse)
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
	_, err = snClient.CreateIncident(wrongIncident)

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

	_, err = snClient.CreateIncident(basicIncident)

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
	_, err = snClient.CreateIncident(basicIncident)

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

	_, err = snClient.CreateIncident(basicIncident)

	if err == nil {
		t.Errorf("Expected an error, got none")
	}
}
