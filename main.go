package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"

	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/yaml.v2"

	"github.com/prometheus/common/log"

	"crypto/md5"
	tmpltext "text/template"
)

var (
	configFile           = kingpin.Flag("config.file", "ServiceNow configuration file.").Default("config/servicenow.yml").String()
	listenAddress        = kingpin.Flag("web.listen-address", "The address to listen on for HTTP requests.").Default(":9877").String()
	config               Config
	serviceNow           ServiceNow
	noUpdateStates       map[json.Number]bool
	incidentUpdateFields map[string]bool
)

// Config - ServiceNow webhook configuration
type Config struct {
	ServiceNow      ServiceNowConfig      `yaml:"service_now"`
	Workflow        WorkflowConfig        `yaml:"workflow"`
	DefaultIncident DefaultIncidentConfig `yaml:"default_incident"`
}

// ServiceNowConfig - ServiceNow instance configuration
type ServiceNowConfig struct {
	InstanceName string `yaml:"instance_name"`
	UserName     string `yaml:"user_name"`
	Password     string `yaml:"password"`
}

// WorkflowConfig - Incident workflow configuration
type WorkflowConfig struct {
	IncidentGroupKeyField string        `yaml:"incident_group_key_field"`
	NoUpdateStates        []json.Number `yaml:"no_update_states"`
	IncidentUpdateFields  []string      `yaml:"incident_update_fields"`
}

// DefaultIncidentConfig - Default configuration for an incident
type DefaultIncidentConfig struct {
	AssignmentGroup  string `yaml:"assignment_group"`
	Category         string `yaml:"category"`
	CmdbCI           string `yaml:"cmdb_ci"`
	Comments         string `yaml:"comments"`
	Company          string `yaml:"company"`
	ContactType      string `yaml:"contact_type"`
	Description      string `yaml:"description"`
	Impact           string `yaml:"impact"`
	ShortDescription string `yaml:"short_description"`
	SubCategory      string `yaml:"subcategory"`
	Urgency          string `yaml:"urgency"`
}

// JSONResponse is the Webhook http response
type JSONResponse struct {
	Status  int
	Message string
}

func webhook(w http.ResponseWriter, r *http.Request) {

	data, err := readRequestBody(r)
	if err != nil {
		log.Errorf("Error reading request body : %v", err)
		sendJSONResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	err = onAlertGroup(data)

	if err != nil {
		log.Errorf("Error managing incident from alert : %v", err)
		sendJSONResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Returns a 200 if everything went smoothly
	sendJSONResponse(w, http.StatusOK, "Success")
}

// Starts 2 listeners
// - first one to give a status on the receiver itself
// - second one to actually process the data
func main() {
	kingpin.Version(version.Print("alertmanager-webhook-servicenow"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	_, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("Error loading config file: %v", err)
	}

	_, err = loadSnClient()
	if err != nil {
		log.Fatalf("Error loading ServiceNow client: %v", err)
	}

	log.Info("Starting webhook", version.Info())
	log.Info("Build context", version.BuildContext())

	http.HandleFunc("/webhook", webhook)
	http.Handle("/metrics", promhttp.Handler())

	log.Infof("listening on: %v", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}

func sendJSONResponse(w http.ResponseWriter, status int, message string) {
	data := JSONResponse{
		Status:  status,
		Message: message,
	}
	bytes, _ := json.Marshal(data)

	w.WriteHeader(status)
	_, err := w.Write(bytes)

	if err != nil {
		log.Errorf("Error writing JSON response: %s", err)
	}
}

func readRequestBody(r *http.Request) (template.Data, error) {

	// Do not forget to close the body at the end
	defer r.Body.Close()

	// Extract data from the body in the Data template provided by AlertManager
	data := template.Data{}
	err := json.NewDecoder(r.Body).Decode(&data)

	return data, err
}

func loadConfig(configFile string) (Config, error) {
	config = Config{}

	// Load the config from the file
	configData, err := ioutil.ReadFile(configFile)
	if err != nil {
		return config, err
	}

	errYAML := yaml.Unmarshal([]byte(configData), &config)
	if errYAML != nil {
		return config, err
	}

	// Load internal state from config
	noUpdateStates = make(map[json.Number]bool, len(config.Workflow.NoUpdateStates))
	for _, s := range config.Workflow.NoUpdateStates {
		noUpdateStates[s] = true
	}

	// Load internal incidents update fields from config
	incidentUpdateFields = make(map[string]bool, len(config.Workflow.IncidentUpdateFields))
	for _, f := range config.Workflow.IncidentUpdateFields {
		incidentUpdateFields[f] = true
	}

	log.Info("ServiceNow config loaded")
	return config, nil
}

func loadSnClient() (ServiceNow, error) {
	var err error
	serviceNow, err = NewServiceNowClient(config.ServiceNow.InstanceName, config.ServiceNow.UserName, config.ServiceNow.Password)
	if err != nil {
		return serviceNow, err
	}
	return serviceNow, nil
}

func onAlertGroup(data template.Data) error {

	log.Infof("Received alert group: Status=%s, GroupLabels=%v, CommonLabels=%v, CommonAnnotations=%v",
		data.Status, data.GroupLabels, data.CommonLabels, data.CommonAnnotations)

	getParams := map[string]string{
		config.Workflow.IncidentGroupKeyField: getGroupKey(data),
	}

	existingIncidents, err := serviceNow.GetIncidents(getParams)
	if err != nil {
		return err
	}

	var existingIncident Incident
	if len(existingIncidents) > 0 {
		existingIncident = existingIncidents[0]

		if len(existingIncidents) > 1 {
			log.Warnf("Found multiple existing incidents for alert group key: %s. Will use first one.", getGroupKey(data))
		}
	}

	if data.Status == "firing" {
		return onFiringGroup(data, existingIncident)
	} else if data.Status == "resolved" {
		return onResolvedGroup(data, existingIncident)
	} else {
		log.Errorf("Unknown alert group status: %s", data.Status)
	}

	return nil
}

func onFiringGroup(data template.Data, existingIncident Incident) error {
	incidentCreateParam, err := alertGroupToIncident(data)
	if err != nil {
		return err
	}

	incidentUpdateParam := filterForUpdate(incidentCreateParam)

	if existingIncident == nil {
		log.Infof("Found no existing incident for firing alert group key: %s", getGroupKey(data))
		if _, err := serviceNow.CreateIncident(incidentCreateParam); err != nil {
			return err
		}
	} else {
		log.Infof("Found existing incident (%s), with state %s, for firing alert group key: %s", existingIncident.GetNumber(), existingIncident.GetState(), getGroupKey(data))
		if noUpdateStates[existingIncident.GetState()] {
			if _, err := serviceNow.CreateIncident(incidentCreateParam); err != nil {
				return err
			}
		} else {
			if _, err := serviceNow.UpdateIncident(incidentUpdateParam, existingIncident.GetSysID()); err != nil {
				return err
			}
		}
	}
	return nil
}

func onResolvedGroup(data template.Data, existingIncident Incident) error {
	incidentCreateParam, err := alertGroupToIncident(data)
	if err != nil {
		return err
	}

	incidentUpdateParam := filterForUpdate(incidentCreateParam)
	if existingIncident == nil {
		log.Errorf("Found no existing incident for resolved alert group key: %s. No incident will be created/updated.", getGroupKey(data))
	} else {
		log.Infof("Found existing incident (%s), with state %s, for resolved alert group key: %s", existingIncident.GetNumber(), existingIncident.GetState(), getGroupKey(data))
		if _, err := serviceNow.UpdateIncident(incidentUpdateParam, existingIncident.GetSysID()); err != nil {
			return err
		}
	}
	return nil
}

func alertGroupToIncident(data template.Data) (Incident, error) {

	incident := Incident{
		"assignment_group":                    config.DefaultIncident.AssignmentGroup,
		"category":                            config.DefaultIncident.Category,
		"contact_type":                        config.DefaultIncident.ContactType,
		"caller_id":                           config.ServiceNow.UserName,
		"cmdb_ci":                             config.DefaultIncident.CmdbCI,
		"comments":                            config.DefaultIncident.Comments,
		"company":                             config.DefaultIncident.Company,
		"description":                         config.DefaultIncident.Description,
		"impact":                              config.DefaultIncident.Impact,
		"short_description":                   config.DefaultIncident.ShortDescription,
		config.Workflow.IncidentGroupKeyField: getGroupKey(data),
		"subcategory":                         config.DefaultIncident.SubCategory,
		"urgency":                             config.DefaultIncident.Urgency,
	}

	applyIncidentTemplate(incident, data)
	validateIncident(incident)

	return incident, nil
}

func filterForUpdate(incident Incident) Incident {
	incidentUpdate := Incident{}
	for field, value := range incident {
		if incidentUpdateFields[field] {
			incidentUpdate[field] = value
		}
	}
	return incidentUpdate
}

func getGroupKey(data template.Data) string {
	hash := md5.Sum([]byte(fmt.Sprintf("%v", data.GroupLabels.SortedPairs())))
	return fmt.Sprintf("%x", hash)
}

func applyIncidentTemplate(incident Incident, data template.Data) error {
	for key, val := range incident {
		var err error
		incident[key], err = applyTemplate(key, val.(string), data)
		if err != nil {
			return err
		}
	}
	return nil
}

func applyTemplate(name string, text string, data template.Data) (string, error) {
	tmpl, err := tmpltext.New(name).Parse(text)
	if err != nil {
		return "", err
	}

	var result bytes.Buffer
	err = tmpl.Execute(&result, data)
	if err != nil {
		return "", err
	}

	return result.String(), nil
}

func validateIncident(incident Incident) error {
	impact := incident["impact"].(string)
	if len(impact) > 0 {
		if _, err := strconv.Atoi(impact); err != nil {
			log.Errorf("'impact' field value is '%v' but should be a digit, please fix your configuration. Incident creation/update will proceed but this field will be missing", impact)
		}
	}

	urgency := incident["urgency"].(string)
	if len(urgency) > 0 {
		if _, err := strconv.Atoi(urgency); err != nil {
			log.Errorf("'urgency' field value is '%v' but should be a digit, please fix your configuration. Incident creation/update will proceed but this field will be missing", urgency)
		}
	}

	return nil
}
