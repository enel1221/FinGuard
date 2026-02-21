BINARY       := finguard
MODULE       := github.com/inelson/finguard
VERSION      ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT       ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME   ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS      := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)
GOFLAGS      := -trimpath
DOCKER_IMAGE ?= finguard
HELM_DIR     := deploy/helm/finguard
PORT         ?= 8080
AIR          := $(shell go env GOPATH)/bin/air

.PHONY: all build test lint clean docker-build docker-push helm-lint helm-template proto run run-all dev frontend swagger

all: lint test build

swagger:
	swag init -g cmd/finguard/main.go -o docs/swagger --parseInternal --exclude headlamp,opencost

build: swagger
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/finguard

run:
	FINGUARD_ADDR=":$(PORT)" \
	FINGUARD_DEV_MODE=true \
	FINGUARD_AUTH_DISABLED=true \
	$(AIR)

run-all:
	docker compose -f docker-compose.dev.yml up -d postgres dex
	@echo "Waiting for dependencies to be healthy..."
	@docker compose -f docker-compose.dev.yml exec postgres sh -c 'until pg_isready -U finguard; do sleep 1; done'
	@echo "Dependencies ready. Starting finguard with live reload..."
	FINGUARD_ADDR=":$(PORT)" \
	FINGUARD_DEV_MODE=true \
	FINGUARD_DB_DSN="postgres://finguard:finguard@localhost:5432/finguard?sslmode=disable" \
	FINGUARD_OIDC_ISSUER="http://localhost:5556" \
	FINGUARD_OIDC_CLIENT_ID="finguard" \
	FINGUARD_OIDC_CLIENT_SECRET="finguard-dev-secret" \
	FINGUARD_OIDC_REDIRECT_URL="http://localhost:$(PORT)/callback" \
	FINGUARD_SESSION_SECRET="dev-session-secret-change-me-32b" \
	FINGUARD_LOG_LEVEL="debug" \
	$(AIR)

dev: build
	FINGUARD_ADDR=":$(PORT)" FINGUARD_DEV_MODE=true FINGUARD_AUTH_DISABLED=true ./bin/$(BINARY)

frontend:
	cd web/frontend && npm ci && npm run build

test:
	go test -race -coverprofile=coverage.out ./...
	cd web/frontend && npm run test

lint:
	golangci-lint run ./...

clean:
	docker compose -f docker-compose.dev.yml down -v --remove-orphans 2>/dev/null || true
	rm -rf bin/ tmp/ coverage.out

docker-build:
	docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) .

docker-push: docker-build
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest

helm-lint:
	helm lint $(HELM_DIR)

helm-template:
	helm template finguard $(HELM_DIR)

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		protos/plugin/plugin.proto
