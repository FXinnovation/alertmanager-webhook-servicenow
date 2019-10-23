package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/prometheus/alertmanager/template"
	"github.com/stretchr/testify/mock"
)

type MockedSnClient struct {
	mock.Mock
}

func (mock *MockedSnClient) CreateIncident(incidentParam Incident) (Incident, error) {
	args := mock.Called(incidentParam)
	return args.Get(0).(Incident), args.Error(1)
}

func (mock *MockedSnClient) GetIncidents(params map[string]string) ([]Incident, error) {
	args := mock.Called(params)
	return args.Get(0).([]Incident), args.Error(1)
}

func (mock *MockedSnClient) UpdateIncident(incidentParam Incident, sysID string) (Incident, error) {
	args := mock.Called(incidentParam, sysID)
	return args.Get(0).(Incident), args.Error(1)
}

func TestLoadSnClient_OK(t *testing.T) {
	loadConfig("config/servicenow_example.yml")
	_, err := loadSnClient()
	if err != nil {
		t.Fatal(err)
	}
}

func TestWebhookHandler_Firing_DoNotExists_OK(t *testing.T) {
	loadConfig("config/servicenow_example.yml")
	incidentUpdateFields = map[string]bool{}
	snClientMock := new(MockedSnClient)
	serviceNow = snClientMock
	snClientMock.On("GetIncidents", mock.Anything).Return([]Incident{}, nil)
	snClientMock.On("CreateIncident", mock.Anything).Run(func(args mock.Arguments) {
		incident := args.Get(0).(Incident)
		if len(incident) == 0 {
			t.Errorf("Wrong incident len: got %v, do not want %v", len(incident), 0)
		}
	}).Return(Incident{}, nil)
	snClientMock.On("UpdateIncident", mock.Anything, mock.Anything).Return(Incident{}, errors.New("Update should not be called"))

	// Load a simple example of a body coming from AlertManager
	data, err := ioutil.ReadFile("test/alertmanager_firing.json")
	if err != nil {
		t.Fatal(err)
	}

	// Create a request to pass to the handler
	req := httptest.NewRequest("GET", "/webhook", bytes.NewReader(data))

	// Create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(webhook)

	// Test the handler with the request and record the result
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Wrong status code: got %v, want %v", status, http.StatusOK)
	}

	expected := `{"Status":200,"Message":"Success"}`
	if rr.Body.String() != expected {
		t.Errorf("Unexpected body: got %v, want %v", rr.Body.String(), expected)
	}
}

func TestWebhookHandler_Firing_Exists_Create_OK(t *testing.T) {
	loadConfig("config/servicenow_example.yml")
	snClientMock := new(MockedSnClient)
	serviceNow = snClientMock
	snClientMock.On("GetIncidents", mock.Anything).Return([]Incident{Incident{"state": "6", "number": "INC42", "sys_id": "42"}}, nil)
	snClientMock.On("CreateIncident", mock.Anything).Return(Incident{}, nil)
	snClientMock.On("UpdateIncident", mock.Anything, mock.Anything).Return(Incident{}, errors.New("Update should not be called"))

	// Load a simple example of a body coming from AlertManager
	data, err := ioutil.ReadFile("test/alertmanager_firing.json")
	if err != nil {
		t.Fatal(err)
	}

	// Create a request to pass to the handler
	req := httptest.NewRequest("GET", "/webhook", bytes.NewReader(data))

	// Create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(webhook)

	// Test the handler with the request and record the result
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Wrong status code: got %v, want %v", status, http.StatusOK)
	}

	expected := `{"Status":200,"Message":"Success"}`
	if rr.Body.String() != expected {
		t.Errorf("Unexpected body: got %v, want %v", rr.Body.String(), expected)
	}
}

func TestWebhookHandler_Firing_Exists_Update_OK(t *testing.T) {
	loadConfig("config/servicenow_example.yml")
	incidentUpdateFields = map[string]bool{
		"comments": true,
	}
	snClientMock := new(MockedSnClient)
	serviceNow = snClientMock
	snClientMock.On("GetIncidents", mock.Anything).Return([]Incident{Incident{"state": "1", "number": "INC42", "sys_id": "42"}}, nil)
	snClientMock.On("CreateIncident", mock.Anything).Return(Incident{}, errors.New("Create should not be called"))
	snClientMock.On("UpdateIncident", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		incident := args.Get(0).(Incident)
		if len(incident) != 1 {
			t.Errorf("Wrong incident len: got %v, want %v", len(incident), 1)
		}
	}).Return(Incident{}, nil)

	// Load a simple example of a body coming from AlertManager
	data, err := ioutil.ReadFile("test/alertmanager_firing.json")
	if err != nil {
		t.Fatal(err)
	}

	// Create a request to pass to the handler
	req := httptest.NewRequest("GET", "/webhook", bytes.NewReader(data))

	// Create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(webhook)

	// Test the handler with the request and record the result
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Wrong status code: got %v, want %v", status, http.StatusOK)
	}

	expected := `{"Status":200,"Message":"Success"}`
	if rr.Body.String() != expected {
		t.Errorf("Unexpected body: got %v, want %v", rr.Body.String(), expected)
	}
}

func TestWebhookHandler_Resolved_DoNotExists_OK(t *testing.T) {
	loadConfig("config/servicenow_example.yml")
	snClientMock := new(MockedSnClient)
	serviceNow = snClientMock
	snClientMock.On("GetIncidents", mock.Anything).Return([]Incident{}, nil)
	snClientMock.On("CreateIncident", mock.Anything).Return(Incident{}, errors.New("Create should not be called"))
	snClientMock.On("UpdateIncident", mock.Anything, mock.Anything).Return(Incident{}, errors.New("Update should not be called"))

	// Load a simple example of a body coming from AlertManager
	data, err := ioutil.ReadFile("test/alertmanager_resolved.json")
	if err != nil {
		t.Fatal(err)
	}

	// Create a request to pass to the handler
	req := httptest.NewRequest("GET", "/webhook", bytes.NewReader(data))

	// Create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(webhook)

	// Test the handler with the request and record the result
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Wrong status code: got %v, want %v", status, http.StatusOK)
	}

	expected := `{"Status":200,"Message":"Success"}`
	if rr.Body.String() != expected {
		t.Errorf("Unexpected body: got %v, want %v", rr.Body.String(), expected)
	}
}

func TestWebhookHandler_Resolved_Exists_OK(t *testing.T) {
	loadConfig("config/servicenow_example.yml")
	snClientMock := new(MockedSnClient)
	serviceNow = snClientMock
	snClientMock.On("GetIncidents", mock.Anything).Return([]Incident{Incident{"state": "7", "number": "INC42", "sys_id": "42"}}, nil)
	snClientMock.On("CreateIncident", mock.Anything).Return(Incident{}, errors.New("Create should not be called"))
	snClientMock.On("UpdateIncident", mock.Anything, mock.Anything).Return(Incident{}, nil)

	// Load a simple example of a body coming from AlertManager
	data, err := ioutil.ReadFile("test/alertmanager_resolved.json")
	if err != nil {
		t.Fatal(err)
	}

	// Create a request to pass to the handler
	req := httptest.NewRequest("GET", "/webhook", bytes.NewReader(data))

	// Create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(webhook)

	// Test the handler with the request and record the result
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Wrong status code: got %v, want %v", status, http.StatusOK)
	}

	expected := `{"Status":200,"Message":"Success"}`
	if rr.Body.String() != expected {
		t.Errorf("Unexpected body: got %v, want %v", rr.Body.String(), expected)
	}
}

func TestWebhookHandler_BadRequest(t *testing.T) {
	loadConfig("config/servicenow_example.yml")

	// Create a request to pass to the handler
	req := httptest.NewRequest("GET", "/webhook", nil)

	// Create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(webhook)

	// Test the handler with the request and record the result
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Wrong status code: got %v, want %v", status, http.StatusBadRequest)
	}

	expected := `{"Status":400,"Message":"EOF"}`
	if rr.Body.String() != expected {
		t.Errorf("Unexpected body: got %v, want %v", rr.Body.String(), expected)
	}
}

func TestWebhookHandler_InternalServerError(t *testing.T) {
	loadConfig("config/servicenow_example.yml")
	snClientMock := new(MockedSnClient)
	serviceNow = snClientMock
	snClientMock.On("GetIncidents", mock.Anything).Return([]Incident{}, nil)
	snClientMock.On("CreateIncident", mock.Anything).Return(Incident{}, errors.New("Error"))

	// Load a simple example of a body coming from AlertManager
	data, err := ioutil.ReadFile("test/alertmanager_firing.json")
	if err != nil {
		t.Fatal(err)
	}

	// Create a request to pass to the handler
	req := httptest.NewRequest("GET", "/webhook", bytes.NewReader(data))

	// Create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(webhook)

	// Test the handler with the request and record the result
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Wrong status code: got %v, want %v", status, http.StatusInternalServerError)
	}

	// Check the response body
	expected := `{"Status":500,"Message":"Error"}`
	if rr.Body.String() != expected {
		t.Errorf("Unexpected body: got %v, want %v", rr.Body.String(), expected)
	}
}

func TestApplyTemplate_emptyText(t *testing.T) {
	data := template.Data{}
	text := ""
	result, err := applyTemplate("name", text, data)
	if err != nil {
		t.Fatal(err)
	}
	expected := ""
	if string(result) != expected {
		t.Errorf("Unexpected result: got %v, want %v", result, expected)
	}
}

func TestApplyTemplate_OK(t *testing.T) {
	data := template.Data{
		Status: "firing",
		CommonAnnotations: map[string]string{
			"error": "my error",
		},
	}
	text := "Status is {{.Status}} and error is {{.CommonAnnotations.error}}"
	result, err := applyTemplate("name", text, data)
	if err != nil {
		t.Fatal(err)
	}
	expected := "Status is firing and error is my error"
	if string(result) != expected {
		t.Errorf("Unexpected result: got %v, want %v", result, expected)
	}
}

func TestApplyIncidentTemplate_Range(t *testing.T) {
	data := template.Data{
		CommonAnnotations: map[string]string{
			"error":   "a",
			"warning": "b",
		},
	}
	incident := Incident{
		"description": "{{ range $key, $val := .CommonAnnotations}}{{ $key }}:{{ $val }} {{end}}",
	}
	applyIncidentTemplate(incident, data)

	result := incident["description"]
	expected := "error:a warning:b "

	if result != expected {
		t.Errorf("Unexpected result: got %v, want %v", result, expected)
	}
}

func TestLoadConfigContent_Ok(t *testing.T) {
	configFile := `
service_now:
 instance_name: "instance"
 user_name: "SA"
 password: "SA!" 

default_incident:
 assignment_group: "1234"
`
	defaultIncident := make(map[string]string)
	defaultIncident["assignment_group"] = "1234"
	goodConfig := Config{
		ServiceNow: ServiceNowConfig{
			InstanceName: "instance",
			UserName:     "SA",
			Password:     "SA!",
		},
		Workflow:        WorkflowConfig{},
		DefaultIncident: defaultIncident,
	}
	config, err := loadConfigContent([]byte(configFile))
	if err != nil {
		t.Errorf("Error parsing viable content %v", err)
	}
	if !reflect.DeepEqual(config, goodConfig) {
		t.Errorf("Error in getting config Got:%v, Expected config:%v", goodConfig, config)
	}
}

func TestLoadConfigContent_ParsingError(t *testing.T) {
	configFile := `
service_now:
 instance_name: "instance"
 user_name: "SA"
 password: "SA!" 
TOTO
:tatata
`

	_, err := loadConfigContent([]byte(configFile))
	if err == nil {
		t.Errorf("Should have an error parsing unparseable content")
	}
}

func Test_validateIncident(t *testing.T) {
	type args struct {
		incident Incident
	}
	tests := []struct {
		name string
		args args
		want []error
	}{
		{
			name: "empty",
			args: args{Incident{}},
			want: []error{},
		},
		{
			name: "good",
			args: args{Incident{"impact": "2", "urgency": "2"}},
			want: []error{},
		},
		{
			name: "nil",
			args: args{Incident{"impact": nil, "urgency": nil}},
			want: []error{},
		},
		{
			name: "empty_string",
			args: args{Incident{"impact": "", "urgency": ""}},
			want: []error{},
		},
		{
			name: "string",
			args: args{Incident{"impact": "<no value>", "urgency": "<no value>"}},
			want: []error{fmt.Errorf("first error"), fmt.Errorf("second error")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateIncident(tt.args.incident); len(got) != len(tt.want) {
				t.Errorf("validateIncident() = %v, want %v", got, tt.want)
			}
		})
	}
}
