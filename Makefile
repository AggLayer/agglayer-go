include version.mk

# Check for required dependencies
CHECK_GO := $(shell command -v go 2> /dev/null)
CHECK_CURL := $(shell command -v curl 2> /dev/null)
CHECK_DOCKER := $(shell command -v docker 2> /dev/null)

check-go:
ifndef CHECK_GO
	$(error "Go is not installed. Please install Go and retry.")
endif

check-curl:
ifndef CHECK_CURL
	$(error "curl is not installed. Please install curl and retry.")
endif

check-docker:
ifndef CHECK_DOCKER
	$(error "Docker is not installed. Please install Docker and retry.")
endif

# Targets that require the checks
build: check-go
build-docker: check-docker
build-docker-nc: check-docker
run-docker: check-docker
run-docker-bo: run-docker-bo
stop-docker: check-docker
destroy-docker: check-docker
install-linter: check-go check-curl
lint: check-go
unit-tests: check-go
e2e-tests: check-go

ARCH := $(shell uname -m)

ifeq ($(ARCH),x86_64)
	ARCH = amd64
else
	ifeq ($(ARCH),aarch64)
		ARCH = arm64
	endif
endif

GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/dist
GOOS := $(shell uname -s  | tr '[:upper:]' '[:lower:]')
GOENVVARS := GOBIN=$(GOBIN) CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(ARCH)
GOBINARY := beethoven
GOCMD := $(GOBASE)/cmd

LDFLAGS += -X 'github.com/0xPolygon/beethoven.Version=$(VERSION)'
LDFLAGS += -X 'github.com/0xPolygon/beethoven.GitRev=$(GITREV)'
LDFLAGS += -X 'github.com/0xPolygon/beethoven.GitBranch=$(GITBRANCH)'
LDFLAGS += -X 'github.com/0xPolygon/beethoven.BuildDate=$(DATE)'

.PHONY: build
build: ## Builds the binary locally into ./dist
	$(GOENVVARS) go build -ldflags "all=$(LDFLAGS)" -o $(GOBIN)/$(GOBINARY) $(GOCMD)

.PHONY: build-docker
build-docker: ## Builds a docker image with the beethoven binary
	docker compose -f ./docker/docker-compose.yaml build beethoven

.PHONY: build-docker-nc
build-docker-nc: ## Builds a docker image with the beethoven binary - but without build cache
	docker compose -f ./docker/docker-compose.yaml build --no-cache beethoven

.PHONY: run-docker
run-docker: ## Builds and runs beethoven with the default list of required services such as l1 and zkevm node
	docker compose -f ./docker/docker-compose.yaml up -d l1 zkevm-prover zkevm-node
	docker compose -f ./docker/docker-compose.yaml up -d --build beethoven

.PHONY: run-docker-bo
run-docker-bo: ## Builds and runs beethoven only
	docker compose -f ./docker/docker-compose.yaml up -d --build beethoven

.PHONY: stop-docker
stop-docker: ## Stops beethoven container and other side services without removing state
	docker compose -f ./docker/docker-compose.yaml stop

.PHONY: destroy-docker
destroy-docker: ## Stops and removes beethoven container and other side services
	docker compose -f ./docker/docker-compose.yaml down

.PHONY: install-linter
install-linter: ## Installs the linter
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.52.2

.PHONY: lint
lint: ## Runs the linter
	export "GOROOT=$$(go env GOROOT)" && $$(go env GOPATH)/bin/golangci-lint run

## Help display.
## Pulls comments from beside commands and prints a nicely formatted
## display with the commands and their usage information.
.DEFAULT_GOAL := help

.PHONY: help
help: ## Prints the help
		@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: e2e-tests
e2e-tests: ## Runs E2E tests
.PHONY: unit-tests
unit-tests: ## Runs unit tests
	go test $(go list ./... | grep -v ./test) -v -timeout=5m
