GO    := GO15VENDOREXPERIMENT=1 go
PROMU := $(GOPATH)/bin/promu
pkgs   = $(shell $(GO) list ./...)

PREFIX                  ?= $(shell pwd)
BIN_DIR                 ?= $(shell pwd)
DOCKER_REPO             ?= fxinnovation
DOCKER_IMAGE_NAME       ?= alertmanager-webhook-servicenow
DOCKER_IMAGE_TAG        ?= $(subst /,-,$(shell git rev-parse --abbrev-ref HEAD))

all: format build test

test: build ## running test after build
	@echo ">> running tests"
	@$(GO) test -v -short $(pkgs)

style: ## check code style
	@echo ">> checking code style"
	@! gofmt -d $(shell find . -path ./vendor -prune -o -name '*.go' -print) | grep '^'

format: ## Format code
	@echo ">> formatting code"
	@$(GO) fmt $(pkgs)

vet: ## vet code
	@echo ">> vetting code"
	@$(GO) vet $(pkgs)

build: ## build code with promu
	@echo ">> building binaries"
	@$(PROMU) build --prefix $(PREFIX)

tarball: ## creates a release tarball
	@echo ">> building release tarball"
	@$(PROMU) tarball --prefix $(PREFIX) $(BIN_DIR)

docker: ## creates docker image
	@echo ">> building docker image"
	@docker build -t "$(DOCKER_REPO)/$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)" .

dockerlint: ## lints dockerfile
	@echo ">> linting Dockerfile"
	@docker run --rm -i hadolint/hadolint < Dockerfile

lint: golint ## lint code
	@echo ">> linting code"
	@! golint $(pkgs) | grep '^'

golint: ## gets golint for building
	@go get -u golang.org/x/lint/golint

getpromu:
	@GOOS=$(shell uname -s | tr A-Z a-z) \
		GOARCH=$(subst x86_64,amd64,$(patsubst i%86,386,$(shell uname -m))) \
		$(GO) get github.com/prometheus/promu

.PHONY: all style format build test vet tarball getpromu

VERSION?=$(shell cat VERSION)
TAGEXISTS=$(shell git tag --list | egrep -q "^$(VERSION)$$" && echo 1 || echo 0)

release-tag: ## create a release tag
	@if [ "$(TAGEXISTS)" -eq "0" ]; then \
		echo ">>Creating tag $(VERSION)";\
		git tag $(VERSION) && git push -u origin $(VERSION);\
	fi

crossbuild: ## creates a crossbuild in .build folder for either all the supported platforms or the one passed as OSARCH var
	@if [ -n "$(OSARCH)" ]; then \
		echo ">> Building $(OSARCH)";\
		$(PROMU) crossbuild -p $(OSARCH);\
	else\
		echo ">> Building all platforms";\
		$(PROMU) crossbuild;\
	fi
