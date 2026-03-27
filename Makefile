SHELL := /usr/bin/env bash

.PHONY: run-agent build-agent run-gateway build-gateway build-raw fmt smoke

run-agent:
	cd agent && AGENT_LISTEN=127.0.0.1:18080 go run ./cmd/agent

build-agent:
	mkdir -p bin
	cd agent && go build -o ../bin/iscsi-agent ./cmd/agent

run-gateway:
	cd web/gateway && GATEWAY_LISTEN=127.0.0.1:8080 AGENT_SOCKET=/run/iscsi-agent/agent.sock go run .

build-gateway:
	mkdir -p bin
	cd web/gateway && go build -o ../../bin/iscsi-web-gateway .

build-raw: build-agent build-gateway
	./scripts/build-raw.sh

smoke:
	bash ./scripts/smoke-min.sh

fmt:
	cd agent && go fmt ./...
	cd web/gateway && go fmt ./...
