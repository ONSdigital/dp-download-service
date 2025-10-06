SHELL=bash
MAIN=dp-download-service

BUILD=build
BUILD_ARCH=$(BUILD)/$(GOOS)-$(GOARCH)
BIN_DIR?=.

export GOOS?=$(shell go env GOOS)
export GOARCH?=$(shell go env GOARCH)

BUILD_TIME=$(shell date +%s)
GIT_COMMIT=$(shell git rev-parse HEAD)
VERSION ?= $(shell git tag --points-at HEAD | grep ^v | head -n 1)

LDFLAGS = -ldflags "-X 'main.BuildTime=$(BUILD_TIME)' -X 'main.GitCommit=$(GIT_COMMIT)' -X 'main.Version=$(VERSION)'"

.PHONY: all
all: audit test build

.PHONY: audit
audit:
	dis-vulncheck

.PHONY: build
build:
	@mkdir -p $(BUILD_ARCH)/$(BIN_DIR)
	go build -o $(BUILD_ARCH)/$(BIN_DIR)/$(MAIN) $(LDFLAGS)

.PHONY: debug
debug: build
	HUMAN_LOG=1 go run $(LDFLAGS) -race main.go

.PHONY: debug-run
debug-run:
	HUMAN_LOG=1 DEBUG=1 go run -race -tags 'debug' $(LDFLAGS) main.go

.PHONY: test
test:
	go test -cover ./...

.PHONY: lint
lint:
	golangci-lint run ./... --timeout 5m --tests=false --skip-dirs=features

docker-test-component:
	docker-compose -f docker-compose.yml down
	docker-compose -f docker-compose.yml up --abort-on-container-exit
	docker-compose -f docker-compose.yml down

docker-local:
	docker-compose -f docker-compose-local.yml down
	docker-compose -f docker-compose-local.yml up -d
	docker-compose -f docker-compose-local.yml exec download-service bash
