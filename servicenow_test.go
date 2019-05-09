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
	ContactType:      "Monitoring System",
	Description:      "This is the description",
	ShortDescription: "This is the short description",
	Impact:           "4",
	Urgency:          "3",
}

func TestNewServiceNowClient_OK(t *testing.T) {
	snClient, err := NewServiceNowClient("instanceName", "userName", "password")

	if err != nil {
		t.Errorf("Error occured %s", err)
	}

	if snClient.baseURL != "https://instanceName.service-now.com" {
		t.Errorf("Wrong baseURL %s", snClient.baseURL)
	}

	if snClient.authHeader != "Basic dXNlck5hbWU6cGFzc3dvcmQ=" {
		t.Errorf("Wrong authHeader %s", snClient.authHeader)
	}

	if reflect.TypeOf(&http.Client{}) != reflect.TypeOf(snClient.client) {
		t.Errorf("Wrong client type %s", reflect.TypeOf(snClient.client))
	}
}

func TestNewServiceNowClient_MissingInstanceName(t *testing.T) {
	_, err := NewServiceNowClient("", "userName", "password")

	if err == nil {
		t.Error("Expected an error")
	}
}

func TestCreateIncident(t *testing.T) {
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"sys_id":"424242","number":"INC42"}`)
	}
	ts := httptest.NewServer(http.HandlerFunc(testHandler))
	defer ts.Close()

	snClient, err := NewServiceNowClient("instancename", "username", "password")
	snClient.baseURL = ts.URL

	if err != nil {
		t.Errorf("Error occured on NewServiceNowClient %s", err)
	}

	response, err := snClient.CreateIncident(basicIncident)

	if err != nil {
		t.Errorf("Error occured on CreateIncident %s", err)
	}

	expectedResponse := `{"sys_id":"424242","number":"INC42"}`
	if response != expectedResponse {
		t.Errorf("Wrong response. %s", response)
	}
}
