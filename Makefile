.PHONY: help dev dev-backend dev-frontend build build-backend build-frontend test lint clean docker-build docker-up docker-down migrate

# Variables
DOCKER_COMPOSE = docker-compose
DOCKER_COMPOSE_DEV = docker-compose -f docker-compose.yml -f docker-compose.dev.yml

# Default target
help:
	@echo "RetroTro - Agile Retrospective Tool"
	@echo ""
	@echo "Available targets:"
	@echo "  dev              Start development environment (backend + frontend + postgres)"
	@echo "  dev-backend      Start only backend in dev mode"
	@echo "  dev-frontend     Start only frontend in dev mode"
	@echo "  build            Build all Docker images"
	@echo "  build-backend    Build backend Docker image"
	@echo "  build-frontend   Build frontend Docker image"
	@echo "  test             Run all tests"
	@echo "  test-backend     Run backend tests"
	@echo "  test-frontend    Run frontend tests"
	@echo "  lint             Run linters"
	@echo "  lint-backend     Run backend linter"
	@echo "  lint-frontend    Run frontend linter"
	@echo "  clean            Clean build artifacts"
	@echo "  docker-up        Start all services with Docker"
	@echo "  docker-down      Stop all Docker services"
	@echo "  docker-build     Build Docker images"
	@echo "  migrate          Run database migrations"
	@echo "  migrate-create   Create new migration (name=<name>)"
	@echo "  helm-deps        Update Helm chart dependencies"
	@echo "  helm-template    Render Helm templates"
	@echo "  helm-install     Install Helm chart (dev)"

# Development
dev:
	$(DOCKER_COMPOSE_DEV) up --build

dev-backend:
	cd backend && go run ./cmd/server

dev-frontend:
	cd frontend && npm run dev

# Build
build: build-backend build-frontend

build-backend:
	cd backend && go build -o bin/server ./cmd/server

build-frontend:
	cd frontend && npm run build

# Docker
docker-build:
	$(DOCKER_COMPOSE) build

docker-up:
	$(DOCKER_COMPOSE) up -d

docker-down:
	$(DOCKER_COMPOSE) down

docker-logs:
	$(DOCKER_COMPOSE) logs -f

docker-ps:
	$(DOCKER_COMPOSE) ps

# Testing
test: test-backend test-frontend

test-backend:
	cd backend && go test -v ./...

test-frontend:
	cd frontend && npm test

test-coverage:
	cd backend && go test -cover -coverprofile=coverage.out ./...
	cd backend && go tool cover -html=coverage.out -o coverage.html

# Linting
lint: lint-backend lint-frontend

lint-backend:
	cd backend && go vet ./...
	cd backend && golangci-lint run

lint-frontend:
	cd frontend && npm run lint

# Database
migrate:
	@echo "Running migrations..."
	docker exec -i retrotro-postgres-1 psql -U retrotro -d retrotro < backend/migrations/000001_init_schema.up.sql

migrate-down:
	@echo "Rolling back migrations..."
	docker exec -i retrotro-postgres-1 psql -U retrotro -d retrotro < backend/migrations/000001_init_schema.down.sql

migrate-create:
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-create name=<migration_name>"; exit 1; fi
	@num=$$(ls backend/migrations/*.up.sql 2>/dev/null | wc -l | tr -d ' '); \
	num=$$((num + 1)); \
	num=$$(printf "%06d" $$num); \
	touch backend/migrations/$${num}_$(name).up.sql backend/migrations/$${num}_$(name).down.sql; \
	echo "Created backend/migrations/$${num}_$(name).up.sql"; \
	echo "Created backend/migrations/$${num}_$(name).down.sql"

# Helm
helm-deps:
	cd helm/retrotro && helm dependency update

helm-template:
	helm template retrotro ./helm/retrotro

helm-install:
	helm install retrotro ./helm/retrotro \
		--set postgresql.auth.password=retrotro \
		--set jwt.secret=dev-secret-change-in-production

helm-upgrade:
	helm upgrade retrotro ./helm/retrotro

helm-uninstall:
	helm uninstall retrotro

# Clean
clean:
	rm -rf backend/bin
	rm -rf frontend/dist
	rm -rf frontend/node_modules
	cd backend && go clean

# Go mod
go-mod-tidy:
	cd backend && go mod tidy

go-mod-download:
	cd backend && go mod download

# NPM
npm-install:
	cd frontend && npm install

npm-update:
	cd frontend && npm update

# Setup
setup: go-mod-download npm-install
	@echo "Project setup complete!"

# Format
format: format-backend format-frontend

format-backend:
	cd backend && go fmt ./...

format-frontend:
	cd frontend && npm run format 2>/dev/null || true

# Generate (if needed for swagger/openapi)
generate:
	cd backend && go generate ./...

.DEFAULT_GOAL := help
