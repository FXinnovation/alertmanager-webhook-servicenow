# alertmanager-webhook-servicenow
[![Build Status](https://travis-ci.org/FXinnovation/alertmanager-webhook-servicenow.svg?branch=master)](https://travis-ci.org/FXinnovation/alertmanager-webhook-servicenow)

A [Prometheus AlertManager](https://github.com/prometheus/alertmanager) webhook receiver that manages [ServiceNow](https://www.servicenow.com) incidents from alerts, written in Go.

## Current features
### ServiceNow authentication
The supported authentication to ServiceNow is through a service account (basic authentication through HTTPS).

### Creation of incident by alert group
One incident is created per distinct group key — as defined by the [`group_by`](https://prometheus.io/docs/alerting/configuration/#<route>) parameter of Alertmanager's `route` configuration section. This avoid spamming ServiceNow with incidents when a huge system failure occurs, and still provide a very flexible mechanism to group alerts in one incident. The ServiceNow field used to hold the group key is configurable through the `incident_group_key_field` property and will contain a hash of the group key.

### Incident management workflow
The supported incident workflow is the following:
- Create a new incident if a firing alert group is currently not associated to an existing incident, or if an associated incident exists but is in a state where update is not allowed (this is configurable in the webhook, but will usually be `closed` or `resolved` state)
- Update an existing incident if it is in a state where update is allowed (same configuration as above in the webhook). Incident fields to be updated is also configurable.

Note that when an incident is updated, configured data fields are updated (description, comments, etc...), but incident state is not changed. In the future, an optional auto-resolve feature may be added to move an incident to `resolved` state when the alert group has a resolved status.

## Planned features
- Support more ServiceNow incident fields
- Provide incident template configuration through a separate file
- Support multiple incident configuration templates

## Getting Started

### Prerequisites
To run this project, you will need a [working Go environment](https://golang.org/doc/install).

### Installing
```bash
go get -u github.com/FXinnovation/alertmanager-webhook-servicenow
```

## Testing
This webhook expects a JSON object from Alertmanager. The format of this JSON is described in the [Alertmanager documentation](https://prometheus.io/docs/alerting/configuration/#<webhook_config>) or, alternatively, in the [Alertmanager GoDoc](https://godoc.org/github.com/prometheus/alertmanager/template#Data).

### Manual testing
To quickly test if the webhook is working, you can start it locally (see `Usage`) and then run:

```bash
curl -H "Content-type: application/json" -X POST \
  -d '{"receiver": "servicenow-receiver-1", "status": "firing", "externalURL":"http://my.url", "alerts": [{"status": "firing", "labels": {"alertname": "TestAlert"}, "annotations":{"summary": "My alert summary", "description": "My description"} }], "groupLabels": {"alertname": "TestAlert"}, "commonAnnotations": {"description": "My description"} }' \
  http://localhost:9877/webhook
```

### Running unit tests
```bash
make test
```

## Usage
```bash
./alertmanager-webhook-servicenow -h
```
## Building
Build the sources with 
```bash
make build
```
**Note**: As this is a go build you can use _GOOS_ and _GOARCH_ environment variables to build for another platform.
### Crossbuilding
The _Makefile_ contains a _crossbuild_ target which builds all the platforms defined in _.promu.yml_ file and puts the files in _.build_ folder. Alternatively you can specify one platform to build with the OSARCH environment variable;
```bash
OSARCH=linux/amd64 make crossbuild
```
## Deployment
The webhook listen on port 9877 by default.

### alertmanager-webhook-servicenow config
The webhook config is done in `config/servicenow.yml`.

All `default_incident` properties supports Go templating with structure defined in [AlertManager documentation](https://prometheus.io/docs/alerting/notifications/#data).
Every key value pair in `default_incident` is sent as an incident field.

An example can be found in [config/servicenow_example.yml](https://github.com/FXinnovation/alertmanager-webhook-servicenow/blob/master/config/servicenow_example.yml)

```
service_now:
  instance_name: "<instance name>"
  user_name: "<user>"
  password: "<password>"

workflow:
  # Name of an existing ServiceNow table field that will be used as a key to uniquely reference an alert group in incident management workflow. 
  # This field must accept a minimum of 32 characters.
  incident_group_key_field: "<incident table field>"
  # ID of the incident states for which existing incident will not be updated. 
  # For firing alert group, it will lead to the creation of a new incident.
  # For resolved alert group, no action will be taken.
  no_update_states: [6,7]
  # Name of the incident fields that should be set when an existing incident is updated
  incident_update_fields:
    - "comments"

# All incidents fields configuration supports Go templating
default_incident:
  # Sysid or name of the assignment group
  assignment_group: "<assignment group>"
  # Sysid or name of the category
  category: "<category name>"
  # Sysid or name of the CMDB configuration item
  cmdb_ci: "<configuration item>"
  # Text of the comments
  comments: "<comments text>"
  # Name of the company
  company: "<company name>"
  # Contact type of the incident
  contact_type : "<contact type>"
  # Text of the description
  description: "<description text>"
  # Impact: Business loss and potential damage (for example, financial, customer, regulation, security, reputation, brand) caused by the incident
  # Common values: 1 (High), 2 (Medium), 3 (Low)
  impact: "<impact value>"
  # Text of the short_description
  short_description: "<short description text>"
  # Sysid or name of the subcategory
  subcategory: "<sub category>"
  # Urgency: Speed at which the business expects the incident to be resolved
  # Common values: 1 (High), 2 (Medium), 3 (Low)
  urgency: "<urgency value>"
```

### AlertManager config
In the AlertManager config (e.g., alertmanager.yml), a `webhook_configs` target the webhook URL, e.g.:

```
global:
  resolve_timeout: 5m

route:
  group_by: ['alertname', 'client']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'servicenow-receiver-1'

receivers:
- name: 'servicenow-receiver-1'
  webhook_configs:
  - url: "http://localhost:9877/webhook"
    send_resolved: true
```


## Docker image
You can build a docker image using:
```bash
make docker
```
The resulting image is named `fxinnovation/alertmanager-webhook-servicenow:{git-branch}`.
It exposes port 9877 and expects the config in /config/servicenow.yml. To configure it, you can bind-mount a config from your host: 

```
$ docker run -p 9877 -v /path/on/host/config/servicenow.yml:/config/servicenow.yml fxinnovation/alertmanager-webhook-servicenow:master
```

## Contributing
Refer to [CONTRIBUTING.md](https://github.com/FXinnovation/alertmanager-webhook-servicenow/blob/master/CONTRIBUTING.md).

## License
Apache License 2.0, see [LICENSE](https://github.com/FXinnovation/alertmanager-webhook-servicenow/blob/master/LICENSE).
