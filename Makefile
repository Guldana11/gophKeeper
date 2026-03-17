SHELL := /bin/bash

BUILD_VERSION ?= v0.0.1
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")

LDFLAGS := -X main.buildVersion=$(BUILD_VERSION) \
           -X main.buildDate=$(BUILD_DATE) \
           -X main.gitCommit=$(GIT_COMMIT)

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64
BIN_DIR := bin

.PHONY: proto build-server build-client build build-all test lint clean cover

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/gophkeeper.proto

build-server:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/server ./cmd/server

build-client:
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/client ./cmd/client

build: build-server build-client

build-all:
	@mkdir -p $(BIN_DIR)
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d/ -f1); \
		arch=$$(echo $$platform | cut -d/ -f2); \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		GOOS=$$os GOARCH=$$arch go build -o $(BIN_DIR)/server-$$os-$$arch$$ext ./cmd/server || exit 1; \
		GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/client-$$os-$$arch$$ext ./cmd/client || exit 1; \
	done

test:
	go test -v -race -count=1 ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BIN_DIR)/ coverage.out