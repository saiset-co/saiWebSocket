ifneq (,$(wildcard ./.env))
    include .env
    export
endif

BINARY_NAME=saiwebsocket
ENV_FILE=.env

GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
NC=\033[0m

.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo "$(GREEN)saiWebSocket$(NC)"
	@echo "$(YELLOW)Available commands:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: env
env: ## Create .env file from .env.example
	@if [ -f ".env" ]; then \
		echo "$(GREEN).env already exists$(NC)"; \
	elif [ -f ".env.example" ]; then \
		cp .env.example .env; \
		echo "$(GREEN).env created from .env.example$(NC)"; \
	else \
		echo "$(RED).env.example not found$(NC)"; \
		exit 1; \
	fi

.PHONY: config
config: env ## Generate saiwebsocket.config from template
	@set -a; . ./$(ENV_FILE); set +a; envsubst < ./config.template > ./saiwebsocket.config
	@echo "$(GREEN)Config generated$(NC)"

.PHONY: build
build: ## Build the binary
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags="-w -s" \
		-o $(BINARY_NAME) \
		./websockets_pro.go
	@echo "$(GREEN)Build complete$(NC)"

.PHONY: run
run: config ## Run locally
	@go run ./websockets_pro.go

.PHONY: up
up: env ## Start with docker-compose
	@docker-compose up -d
	@echo "$(GREEN)Started$(NC)"

.PHONY: down
down: ## Stop services
	@docker-compose down
	@echo "$(GREEN)Stopped$(NC)"

.PHONY: restart
restart: ## Rebuild and restart
	@docker-compose build --no-cache
	@docker-compose down
	@docker-compose up -d
	@echo "$(GREEN)Restarted$(NC)"

.PHONY: logs
logs: ## Show logs
	@docker-compose logs -f

.PHONY: status
status: ## Show service status
	@docker-compose ps

.PHONY: clean
clean: ## Clean build artifacts
	@rm -f $(BINARY_NAME) saiwebsocket.config
	@echo "$(GREEN)Cleaned$(NC)"

.PHONY: clean-docker
clean-docker: ## Clean Docker resources
	@docker-compose down -v --remove-orphans
	@echo "$(GREEN)Docker cleaned$(NC)"
