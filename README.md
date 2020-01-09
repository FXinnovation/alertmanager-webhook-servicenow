# alertmanager-webhook-servicenow

[![Build
Status](https://travis-ci.org/FXinnovation/alertmanager-webhook-servicenow.svg?branch=master)](https://travis-ci.org/FXinnovation/alertmanager-webhook-servicenow)

A [Prometheus AlertManager](https://github.com/prometheus/alertmanager) webhook
receiver that manages [ServiceNow](https://www.servicenow.com) incidents from
alerts, written in Go.

## ServiceNow Prerequisites

- A service account with permissions to read and update incidents.
- An available incident table field (minimum of 32 characters) that will be
  dedicated to hold the webhook alert group ID

## Current features

### ServiceNow authentication

The supported authentication to ServiceNow is through a service account (basic
authentication through HTTPS).

### Creation of incident by alert group

One incident is created per distinct group key — as defined by the
[`group_by`](https://prometheus.io/docs/alerting/configuration/#<route>)
parameter of Alertmanager's `route` configuration section. This avoid spamming
ServiceNow with incidents when a huge system failure occurs, and still provide a
very flexible mechanism to group alerts in one incident. The ServiceNow field
used to hold the group key is configurable through the
`incident_group_key_field` property and will contain a hash of the group key.

### Incident management workflow

The supported incident workflow is the following:

- Create a new incident if a firing alert group is currently not associated to
  an existing incident, or if an associated incident exists but is in a state
  where update is not allowed (this is configurable in the webhook, but would
  usually be `resolved`, `closed` and `cancelled` states)
- Update an existing incident if it is in a state where update is allowed (same
  configuration as above in the webhook). Incident fields to be updated is also
  configurable.

Note that when an incident is updated, configured data fields are updated (e.g.:
comments), but incident state is not changed. In the future, an optional
auto-resolve feature may be added to move an incident to `resolved` state when
the alert group has a resolved status.

## Planned features

- Provide incident template configuration through a separate file
- Support multiple incident configuration templates

## Getting Started

### Prerequisites

To run this project from sources, you will need a [working Go
environment](https://golang.org/doc/install).

### Installing

```bash
go get -u github.com/FXinnovation/alertmanager-webhook-servicenow
```

## Building

Build the sources with

```bash
make build
```

**Note**: As this is a Go build you can use _GOOS_ and _GOARCH_ environment
variables to build for another platform.

### Crossbuilding

The _Makefile_ contains a _crossbuild_ target which builds all the platforms
defined in _.promu.yml_ file and puts the files in _.build_ folder.
Alternatively you can specify one platform to build with the OSARCH environment
variable;

```bash
OSARCH=linux/amd64 make crossbuild
```

## Run the binary

```bash
./alertmanager-webhook-servicenow
```

By default, the webhook config is expected in `config/servicenow.yml` (see
[Configuration](#configuration)).

Use `-h` flag to list available options.

## Testing

This webhook expects a JSON object from Alertmanager. The format of this JSON is
described in the [Alertmanager
documentation](https://prometheus.io/docs/alerting/configuration/#<webhook_config>)
or, alternatively, in the [Alertmanager
GoDoc](https://godoc.org/github.com/prometheus/alertmanager/template#Data).

### Manual testing

To quickly test if the webhook is working, first start the binary (see `Run the
binary`). You can then simulate the AlertManager request with cURL:

```bash
curl -H "Content-type: application/json" -X POST \
  -d '{"receiver": "servicenow-receiver-1", "status": "firing", "externalURL":"http://my.url", "alerts": [{"status": "firing", "labels": {"alertname": "TestAlert"}, "annotations":{"summary": "My alert summary", "description": "My description"} }], "groupLabels": {"alertname": "TestAlert"}, "commonAnnotations": {"description": "My description"} }' \
  http://localhost:9877/webhook
```

The first time this command is run, it will create an incident in ServiceNow.
Any additionnal run of this command (with the same `groupLabels`) will update
the existing incident.

### Running unit tests

```bash
make test
```

## Configuration

### alertmanager-webhook-servicenow config

Configuration is usually done in `config/servicenow.yml`.

All `default_incident` properties supports Go templating with the structure
defined in [AlertManager
documentation](https://prometheus.io/docs/alerting/notifications/#data).

An example can be found in
[config/servicenow_example.yml](https://github.com/FXinnovation/alertmanager-webhook-servicenow/blob/master/config/servicenow_example.yml).
Here is the config detailed description:

```yaml
service_now:
  # Mandatory. The instance_name part (subdomain) of your ServiceNow URL (i.e: https://instance_name.service-now.com/)
  instance_name: "<instance name>"
  # Mandatory. A user with permissions to read and update ServiceNow incidents.
  user_name: "<user>"
  password: "<password>"

workflow:
  # Mandatory. Name of an existing ServiceNow incident field that will be used to hold the hashed key that uniquely reference an alert group in the incident management workflow.
  # This field must accept a minimum of 32 characters. A standard approach would be to add a custom field to your incident table (e.g.: u_prometheus_alertgroup_id), and reference it here.
  incident_group_key_field: "<incident table field>"
  # Optional. List of the incident states ID for which existing incident will not be updated. 
  # When the update comes from a firing alert group, it will lead to the creation of a new incident, for resolved alert group, no action will be taken.
  # Usual states configuration would be: resolved, closed and cancelled (e.g. : [6,7,8])
  no_update_states: [6,7,8]
  # Optional. List of incident fields that will be sent to ServiceNow when an existing incident is updated
  # A usual field to set on update would be "comments"
  incident_update_fields: ["comments"]

# All incident fields are optional. The following list is not exhaustive and is provided as an example. Any other existing ServiceNow incident fields are dynamically supported by the webhook, and can be added here
# All incident fields values supports Go templating
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

In the AlertManager config (e.g., alertmanager.yml), a `webhook_configs` target
the webhook URL, e.g.:

```yaml
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

You can run images published in [dockerhub](https://hub.docker.com/r/fxinnovation/alertmanager-webhook-servicenow).

You can also build a docker image using:

```bash
make docker
```

The resulting image is named
`fxinnovation/alertmanager-webhook-servicenow:{git-branch}`.

The image exposes port
9877 and expects the config in `/config/servicenow.yml`. By default,
[servicenow_example.yml](config/servicenow_example.yml) will be placed at
`/config/servicenow.yml`, but it can be overridden by bind-mounting your own
config as shown:

```bash
docker run -p 9877:9877 -v /path/on/host/config/servicenow.yml:/config/servicenow.yml fxinnovation/alertmanager-webhook-servicenow:master
```

The image also accepts environment variables to configure the ServiceNow
connection. If they are present, they will take precedence over the
corresponding variables in the `servicenow.yml` config file:

| Environment Variable                | Corresponding Config Variable                    |
| ----------------------------------- | ------------------------------------------------ |
| SERVICENOW_INSTANCE_NAME            | service_now.instance_name                        |
| SERVICENOW_USERNAME                 | service_now.user_name                            |
| SERVICENOW_PASSWORD                 | service_now.password                             |
| SERVICENOW_INCIDENT_GROUP_KEY_FIELD | workflow.incident_group_key_field                |

Example with environment variables:

```bash
docker run -p 9877:9877 -e SERVICENOW_USERNAME="snow_user" -e SERVICENOW_PASSWORD="snow_password" fxinnovation/alertmanager-webhook-servicenow:master
```

## Contributing

Refer to
[CONTRIBUTING.md](https://github.com/FXinnovation/alertmanager-webhook-servicenow/blob/master/CONTRIBUTING.md).

## License

Apache License 2.0, see
[LICENSE](https://github.com/FXinnovation/alertmanager-webhook-servicenow/blob/master/LICENSE).
