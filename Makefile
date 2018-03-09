SHELL=bash
MAIN=dp-download-service

BUILD=build
BUILD_ARCH=$(BUILD)/$(GOOS)-$(GOARCH)
BIN_DIR?=.

export GOOS?=$(shell go env GOOS)
export GOARCH?=$(shell go env GOARCH)

VAULT_ADDR?='http://127.0.0.1:8200'

# The following variables are used to generate a vault token for the app. The reason for declaring variables, is that
# its difficult to move the token code in a Makefile action. Doing so makes the Makefile more difficult to
# read and starts introduction if/else statements.
VAULT_POLICY:="$(shell vault policy write -address=$(VAULT_ADDR) read-psk policy.hcl)"
TOKEN_INFO:="$(shell vault token create -address=$(VAULT_ADDR) -policy=read-psk -period=24h -display-name=dp-download-service)"
APP_TOKEN:="$(shell echo $(TOKEN_INFO) | awk '{print $$6}')"

build:
	@mkdir -p $(BUILD_ARCH)/$(BIN_DIR)
	go build -o $(BUILD_ARCH)/$(BIN_DIR)/$(MAIN)

debug: build
	HUMAN_LOG=1 VAULT_TOKEN=$(APP_TOKEN) VAULT_ADDR=$(VAULT_ADDR) go run -race main.go

acceptance:
	HUMAN_LOG=1 VAULT_TOKEN=$(APP_TOKEN) VAULT_ADDR=$(VAULT_ADDR) go run main.go

test:
	go test -cover $(shell go list ./... | grep -v /vendor/)

vault:
	@echo "$(VAULT_POLICY)"
	@echo "$(TOKEN_INFO)"
	@echo "$(APP_TOKEN)"

.PHONY: vault debug build test