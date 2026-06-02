SHELL := /bin/sh
DATA_DIR ?= $(HOME)/.duku-net-lab
BIN_DIR ?= $(CURDIR)/bin
CONTAINER_RUNTIME ?= podman
COMPOSE ?= podman-compose

.PHONY: bootstrap configure-local-env up down logs status build test test-go test-dashboard test-swift test-scripts test-postgres-integration lint security-check demo helper-build helper-install helper-uninstall app-build harness schema-check

bootstrap:
	mkdir -p "$(DATA_DIR)/staging" "$(DATA_DIR)/authorized" "$(DATA_DIR)/exports" "$(DATA_DIR)/backups" "$(BIN_DIR)"
	test -f .env || cp .env.example .env
	chmod 600 .env

configure-local-env: bootstrap
	scripts/configure-local-env.sh .env

up: bootstrap
	scripts/check-local-secrets.sh .env
	podman machine start >/dev/null 2>&1 || true
	$(COMPOSE) -f compose.yml up -d --build

down:
	$(COMPOSE) -f compose.yml down

logs:
	$(COMPOSE) -f compose.yml logs -f

status:
	$(COMPOSE) -f compose.yml ps
	curl -fsS http://127.0.0.1:8080/api/v1/status

build:
	$(CONTAINER_RUNTIME) build -f docker/api.Dockerfile -t duku-net-lab-api .
	$(CONTAINER_RUNTIME) build -f docker/worker.Dockerfile -t duku-net-lab-worker .
	$(CONTAINER_RUNTIME) build -f dashboard/Dockerfile -t duku-net-lab-dashboard dashboard
	$(MAKE) helper-build
	$(MAKE) app-build

test: test-go test-dashboard test-swift test-scripts

test-go:
	scripts/go.sh test ./...

test-dashboard:
	cd dashboard && npm test

test-swift:
	cd macos/DukuNetLabMenu && swift test

test-scripts:
	sh -n scripts/*.sh
	scripts/check-loopback.sh
	scripts/check-repository-safety.sh
	scripts/test-local-secrets.sh

test-postgres-integration:
	scripts/test-postgres-integration.sh

lint:
	GOROOT="$$(scripts/go.sh env GOROOT)"; test -z "$$($$GOROOT/bin/gofmt -l cmd internal helper)"
	cd dashboard && npm run build
	scripts/check-loopback.sh

security-check:
	scripts/check-repository-safety.sh

demo:
	curl -fsS -X POST http://127.0.0.1:8080/api/v1/demo/seed

helper-build:
	mkdir -p "$(BIN_DIR)"
	scripts/go.sh build -o bin/duku-capture-helper ./helper

helper-install: helper-build
	sudo scripts/install-helper.sh "$(BIN_DIR)/duku-capture-helper"

helper-uninstall:
	sudo scripts/uninstall-helper.sh

app-build:
	scripts/package-menu-app.sh

harness:
	scripts/go.sh run ./cmd/harness -task fixtures/tasks/example-dashboard.json

schema-check:
	test -s db/init/001_schema.sql
