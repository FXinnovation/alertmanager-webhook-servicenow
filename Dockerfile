FROM golang:1.12 as builder
WORKDIR /go/src/github.com/FXinnovation/alertmanager-webhook-servicenow
COPY . .
RUN make build

FROM quay.io/prometheus/busybox:glibc AS app
LABEL maintainer="FXinnovation CloudToolDevelopment <CloudToolDevelopment@fxinnovation.com>"
COPY --from=builder /go/src/github.com/FXinnovation/alertmanager-webhook-servicenow/alertmanager-webhook-servicenow /bin/alertmanager-webhook-servicenow
COPY --from=builder /go/src/github.com/FXinnovation/alertmanager-webhook-servicenow/config/servicenow_example.yml /config/servicenow.yml

VOLUME [ "/config" ]
EXPOSE      9877
WORKDIR /
ENTRYPOINT  [ "/bin/alertmanager-webhook-servicenow" ]
