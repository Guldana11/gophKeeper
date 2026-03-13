BUILD_VERSION ?= v0.0.1
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X main.buildVersion=$(BUILD_VERSION) -X main.buildDate=$(BUILD_DATE)

.PHONY: proto build-server build-client build test lint clean

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/gophkeeper.proto

build-server:
	go build -o bin/server ./cmd/server

build-client:
	go build -ldflags "$(LDFLAGS)" -o bin/client ./cmd/client

build: build-server build-client

test:
	go test -v -race -count=1 ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/ coverage.out
