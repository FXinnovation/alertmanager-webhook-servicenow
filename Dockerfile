FROM golang:1.12 as builder
WORKDIR /go/src/github.com/FXinnovation/alertmanager-webhook-servicenow
COPY . .
RUN make

FROM ubuntu:18.04
COPY --from=builder /go/src/github.com/FXinnovation/alertmanager-webhook-servicenow/alertmanager-webhook-servicenow /alertmanager-webhook-servicenow
COPY --from=builder /go/src/github.com/FXinnovation/alertmanager-webhook-servicenow/config/servicenow_example.yml /config/servicenow.yml
ADD ./resources /resources
RUN /resources/build && rm -rf /resources
USER aws
VOLUME [ "/config" ]
EXPOSE 9877
WORKDIR /
ENTRYPOINT  [ "/opt/alertmanager-webhook-servicenow/alertmanager-webhook-servicenow" ]

LABEL maintainer="FXinnovation CloudToolDevelopment <CloudToolDevelopment@fxinnovation.com>" \
      "org.label-schema.name"="alertmanager-webhook-servicenow" \
      "org.label-schema.base-image.name"="docker.io/library/ubuntu" \
      "org.label-schema.base-image.version"="18.04" \
      "org.label-schema.description"="alertmanager-webhook-servicenow in a container" \
      "org.label-schema.url"="https://github.com/FXinnovation/alertmanager-webhook-servicenow" \
      "org.label-schema.vcs-url"="https://github.com/FXinnovation/alertmanager-webhook-servicenow" \
      "org.label-schema.vendor"="FXinnovation" \
      "org.label-schema.schema-version"="1.0.0-rc.1" \
      "org.label-schema.usage"="Please see README.md"