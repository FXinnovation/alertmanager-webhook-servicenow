# alertmanager-webhook-servicenow
[![Build Status](https://travis-ci.org/FXinnovation/alertmanager-webhook-servicenow.svg?branch=master)](https://travis-ci.org/FXinnovation/alertmanager-webhook-servicenow)

**WARNING - Work In Progress - this webhook receiver is not ready to be used** 

A [Prometheus AlertManager](https://github.com/prometheus/alertmanager) webhook receiver that manages [ServiceNow](https://www.servicenow.com) incidents from alerts, written in Go.


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
To quickly test if the webhook is working you can run:

```bash
$ curl -H "Content-type: application/json" -X POST \
  -d '{"receiver": "servicenow-receiver-1", "status": "firing", "alerts": [{"status": "firing", "labels": {"alertname": "TestAlert", "key": "value"} }], "groupLabels": {"alertname": "TestAlert"}}' \
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

## Deployment

The webhook listen on port 9877 by default.

## Docker image

You can build a docker image using:
```bash
make docker
```
The resulting image is named `fxinnovation/alertmanager-webhook-servicenow:{git-branch}`.
It exposes port 9877. To configure it, run:
```
$ docker run -p 9877 fxinnovation/alertmanager-webhook-servicenow:master
```

## Contributing

Refer to [CONTRIBUTING.md](https://github.com/FXinnovation/alertmanager-webhook-servicenow/blob/master/CONTRIBUTING.md).

## License

Apache License 2.0, see [LICENSE](https://github.com/FXinnovation/alertmanager-webhook-servicenow/blob/master/LICENSE).
