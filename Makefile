.PHONY: help setup dev build test clean docker-build docker-run

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Install dependencies
	@echo "Installing Go dependencies..."
	go mod download
	@echo "Installing frontend dependencies..."
	cd frontend && npm install

dev-backend: ## Run backend in development mode
	@echo "Starting backend server..."
	go run cmd/api/main.go

dev-frontend: ## Run frontend in development mode
	@echo "Starting frontend development server..."
	cd frontend && npm run dev

dev: ## Run both backend and frontend (requires two terminals)
	@echo "Start backend in one terminal: make dev-backend"
	@echo "Start frontend in another terminal: make dev-frontend"

build-backend: ## Build backend binary
	@echo "Building backend..."
	go build -o api ./cmd/api

build-frontend: ## Build frontend for production
	@echo "Building frontend..."
	cd frontend && npm run build

build: build-backend build-frontend ## Build both backend and frontend

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f api
	rm -rf frontend/dist
	rm -rf frontend/node_modules

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t volunteer-media:latest .

docker-run: ## Run application with Docker Compose
	@echo "Starting application with Docker Compose..."
	docker compose up -d

docker-stop: ## Stop Docker containers
	@echo "Stopping Docker containers..."
	docker compose down

docker-logs: ## View Docker logs
	docker compose logs -f

db-start: ## Start PostgreSQL database
	@echo "Starting PostgreSQL..."
	docker compose up -d postgres_dev

db-stop: ## Stop PostgreSQL database
	docker compose stop postgres_dev

db-shell: ## Connect to PostgreSQL shell
	docker exec -it volunteer_media_db_dev psql -U postgres -d volunteer_media_dev

migrate: ## Run database migrations (automatically runs with dev-backend)
	@echo "Migrations run automatically when starting the backend"

lint: ## Run linter
	@echo "Running golangci-lint..."
	golangci-lint run

fmt: ## Format code
	@echo "Formatting Go code..."
	go fmt ./...
	@echo "Formatting frontend code..."
	cd frontend && npm run lint -- --fix

all: setup build ## Setup and build everything
