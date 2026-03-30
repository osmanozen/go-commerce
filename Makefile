.PHONY: help dev up down lint test build clean

# ─── Variables ────────────────────────────────────────────────────────────────
SERVICES := catalog cart ordering inventory profiles reviews wishlists coupons
COMPOSE_INFRA := -f infra/docker-compose.infra.yml
COMPOSE_SERVICES := -f infra/docker-compose.services.yml

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ─── Infrastructure ──────────────────────────────────────────────────────────
up: ## Start all infrastructure (Kafka, Postgres, Redis)
	docker compose $(COMPOSE_INFRA) up -d
	@echo "Waiting for Kafka to be healthy..."
	@docker compose $(COMPOSE_INFRA) exec -T kafka bash -c 'until kafka-topics --bootstrap-server localhost:9092 --list 2>/dev/null; do sleep 2; done'
	@echo "Infrastructure ready!"

down: ## Stop all infrastructure
	docker compose $(COMPOSE_INFRA) $(COMPOSE_SERVICES) down

# ─── Build ────────────────────────────────────────────────────────────────────
build: $(addprefix build-,$(SERVICES)) ## Build all services


build-%: ## Build a specific service (e.g., make build-catalog)
	cd services/$* && go build -o ../../bin/$* ./cmd/server/

# ─── Development ──────────────────────────────────────────────────────────────
run-%: ## Run a specific service (e.g., make run-catalog)
	cd services/$* && go run ./cmd/server/

# ─── Testing ──────────────────────────────────────────────────────────────────
test: $(addprefix test-,$(SERVICES)) ## Run all tests
	cd pkg/buildingblocks && go test ./... -v -race -count=1

test-%: ## Test a specific service (e.g., make test-catalog)
	cd services/$* && go test ./... -v -race -count=1

# ─── Code Quality ─────────────────────────────────────────────────────────────
lint: ## Run linters
	golangci-lint run ./...

fmt: ## Format all Go files
	gofmt -s -w .

vet: ## Run go vet
	go vet ./...

# ─── Docker ───────────────────────────────────────────────────────────────────
docker-build-%: ## Build Docker image for a service (e.g., make docker-build-catalog)
	docker build --build-arg SERVICE=$* -f infra/docker/Dockerfile.service -t go-commerce-$*:latest .

docker-build-all: $(addprefix docker-build-,$(SERVICES)) ## Build Docker images for all services


# ─── Housekeeping ─────────────────────────────────────────────────────────────
clean: $(addprefix clean-,$(SERVICES)) ## Remove build artifacts
	rm -rf bin/

clean-%:
	rm -f services/$*/$*

tidy: $(addprefix tidy-,$(SERVICES)) ## Run go mod tidy for all modules
	cd pkg/buildingblocks && go mod tidy

tidy-%:
	cd services/$* && go mod tidy
