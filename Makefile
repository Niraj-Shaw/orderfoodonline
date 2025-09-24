# -------------------------------
# Food Ordering API - Makefile
# -------------------------------

.PHONY: help build run run-bin test tidy fmt vet clean deps \
        docker-build docker-run docker-push check-coupons verify dev-setup

# ------- Config -------
APP           ?= orderfoodonline
PKG_MAIN      ?= ./cmd/server
BIN_DIR       ?= bin
BIN           ?= $(BIN_DIR)/$(APP)

PORT          ?= 8080
SERVER_ADDR   ?= :$(PORT)
API_KEY       ?= apitest
COUPON_DIR    ?= ./data

# Docker
IMG           ?= $(APP):dev
PLATFORM      ?= linux/amd64
REGISTRY      ?=

# Build info (optional)
GIT_COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE    := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS       := -s -w -X main.gitCommit=$(GIT_COMMIT) -X main.buildDate=$(BUILD_DATE)

# ------- Help -------
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ------- Go basics -------
deps: ## Install / update go deps
	@go mod download
	@go mod tidy

fmt: ## Format code
	@go fmt ./...

vet: ## Run go vet
	@go vet ./...

test: ## Run tests (with race detector)
	@go test ./... -count=1 -race -v

# ------- Build / Run -------
build: ## Build the application binary
	@echo "Building $(BIN)…"
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) $(PKG_MAIN)
	@ls -lh $(BIN)

run: ## Run the application from source
	@echo "Starting $(APP) on $(SERVER_ADDR)"
	@SERVER_ADDR=$(SERVER_ADDR) API_KEY=$(API_KEY) COUPON_DIR=$(COUPON_DIR) \
	go run $(PKG_MAIN)

run-bin: build ## Build then run the compiled binary
	@SERVER_ADDR=$(SERVER_ADDR) API_KEY=$(API_KEY) COUPON_DIR=$(COUPON_DIR) \
	./$(BIN)

clean: ## Clean build artifacts
	@rm -rf $(BIN_DIR)
	@go clean

# ------- Utilities -------
check-coupons: ## Check for required coupon files in $(COUPON_DIR)
	@echo "Checking $(COUPON_DIR) for coupon files…"
	@for f in couponbase1.gz couponbase2.gz couponbase3.gz; do \
		if [ -f "$(COUPON_DIR)/$$f" ]; then \
			echo "✅ $(COUPON_DIR)/$$f found"; \
		else \
			echo "❌ $(COUPON_DIR)/$$f missing"; \
		fi; \
	done

dev-setup: ## One-time local setup
	@$(MAKE) deps
	@chmod +x scripts/test_api.sh 2>/dev/null || true
	@echo "Dev setup complete."

# ------- API smoke (requires server running) -------
test-api: ## Test API endpoints (server must be running)
	@./scripts/test_api.sh

verify: build check-coupons ## Build, check coupons, and quick API probe
	@echo "Starting temp server…"
	@( SERVER_ADDR=$(SERVER_ADDR) API_KEY=$(API_KEY) COUPON_DIR=$(COUPON_DIR) ./$(BIN) & echo $$! > .tmp.pid ) ; \
	sleep 2 ; \
	if curl -fsS "http://localhost:$(PORT)/healthz" >/dev/null ; then \
		echo "Healthcheck OK"; \
	else \
		echo "Healthcheck FAILED"; \
		kill $$(cat .tmp.pid) 2>/dev/null || true ; rm -f .tmp.pid ; exit 1 ; \
	fi ; \
	./scripts/test_api.sh || true ; \
	kill $$(cat .tmp.pid) 2>/dev/null || true ; rm -f .tmp.pid ; \
	echo "Verify complete."

# ------- Docker -------
docker-build: ## Build Docker image
	@docker build --platform=$(PLATFORM) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(IMG) .

docker-run: docker-build ## Run Docker container (mounts ./data)
	@docker run --rm -p $(PORT):8080 \
		-e API_KEY=$(API_KEY) \
		-e COUPON_DIR=/data \
		-v "$(PWD)/data:/data:ro" \
		$(IMG)

docker-push: ## Tag and push image (set REGISTRY=repo/image:tag)
	@if [ -z "$(REGISTRY)" ]; then echo "Set REGISTRY=your-registry/image:tag"; exit 1; fi
	@docker tag $(IMG) $(REGISTRY)
	@docker push $(REGISTRY)