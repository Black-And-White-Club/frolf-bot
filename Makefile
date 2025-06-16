.PHONY: migrate-init migrate migrate-all rollback-all run
.PHONY: test-unit-all test-integration-all test-all-project test-all-verbose
.PHONY: test-count-unit test-count-integration test-count-all coverage-all-with-counts
.PHONY: integration-leaderboard-service integration-leaderboard-handlers
.PHONY: integration-user-service integration-user-handlers integration-score-service integration-score-handlers
.PHONY: integration-round-service integration-round-handlers
.PHONY: build-coverage coverage-all coverage-html coverage-unit coverage-integration clean-coverage
.PHONY: mocks-user mocks-leaderboard mocks-round mocks-score mocks-eventbus mocks-all build-version
.PHONY: river-migrate-up river-migrate-down river-clean clean-all
.PHONY: db-config db-test ci-setup help

# --- Database Migration and Run Targets ---
migrate-init:
	go run cmd/bun/main.go migrate init

migrate:
	go run cmd/bun/main.go migrate migrate

clean-all: river-clean rollback-all

# Run River migrations first, then our app migrations
migrate-all: river-migrate-up migrate-init migrate

rollback-all: 
	@echo "Rolling back application migrations..."
	go run cmd/bun/main.go migrate rollback

# Database configuration - can be overridden via environment variables
# Default to loading from .env file if DATABASE_URL is not set
DB_URL ?= $(shell [ -f .env ] && grep '^DATABASE_URL=' .env | cut -d '=' -f2- | tr -d '"' || echo "")
ifeq ($(DB_URL),)
    $(error DATABASE_URL not found. Please set DATABASE_URL environment variable or create .env file with DATABASE_URL)
endif

# Parse DATABASE_URL for psql components (for river-clean)
DB_PARAMS := $(shell echo '$(DB_URL)' | sed -E 's|postgres://([^:]+):([^@]+)@([^:]+):([^/]+)/([^?]+).*|\1 \2 \3 \4 \5|')
DB_USER := $(word 1, $(DB_PARAMS))
DB_PASS := $(word 2, $(DB_PARAMS))
DB_HOST := $(word 3, $(DB_PARAMS))
DB_PORT := $(word 4, $(DB_PARAMS))
DB_NAME := $(word 5, $(DB_PARAMS))

# River migration targets
river-migrate-up:
	@echo "Running River migrations..."
	@echo "Using database: $(DB_HOST):$(DB_PORT)/$(DB_NAME)"
	@if ! command -v river >/dev/null 2>&1; then \
		echo "Installing River CLI..."; \
		go install github.com/riverqueue/river/cmd/river@latest; \
	fi
	river migrate-up --line main --database-url "$(DB_URL)"

river-migrate-down:
	@echo "Rolling back River migrations..."
	@echo "Using database: $(DB_HOST):$(DB_PORT)/$(DB_NAME)"
	@if command -v river >/dev/null 2>&1; then \
		river migrate-down --line main --database-url "$(DB_URL)" --max-steps 10; \
	else \
		echo "River CLI not found, skipping River migration rollback"; \
	fi

# Clean up any leftover River artifacts from manual creation
river-clean:
	@echo "Cleaning up any manual River table artifacts..."
	@echo "Using database: $(DB_HOST):$(DB_PORT)/$(DB_NAME)"
	@go run cmd/bun/main.go migrate rollback || true
	@echo "Dropping any existing River tables and types..."
	@PGPASSWORD="$(DB_PASS)" psql -h "$(DB_HOST)" -p "$(DB_PORT)" -U "$(DB_USER)" -d "$(DB_NAME)" -c "DROP TABLE IF EXISTS river_migration CASCADE;" || true
	@PGPASSWORD="$(DB_PASS)" psql -h "$(DB_HOST)" -p "$(DB_PORT)" -U "$(DB_USER)" -d "$(DB_NAME)" -c "DROP TABLE IF EXISTS river_queue CASCADE;" || true  
	@PGPASSWORD="$(DB_PASS)" psql -h "$(DB_HOST)" -p "$(DB_PORT)" -U "$(DB_USER)" -d "$(DB_NAME)" -c "DROP TABLE IF EXISTS river_leader CASCADE;" || true
	@PGPASSWORD="$(DB_PASS)" psql -h "$(DB_HOST)" -p "$(DB_PORT)" -U "$(DB_USER)" -d "$(DB_NAME)" -c "DROP TABLE IF EXISTS river_job CASCADE;" || true
	@PGPASSWORD="$(DB_PASS)" psql -h "$(DB_HOST)" -p "$(DB_PORT)" -U "$(DB_USER)" -d "$(DB_NAME)" -c "DROP TYPE IF EXISTS river_job_state CASCADE;" || true
	@echo "River cleanup completed"


run:
	go run cmd/app/main.go

# --- Project-Wide Test Targets ---
# Run all unit tests across the entire project
test-unit-all:
	@echo "Running all unit tests across the project..."
	go test ./app/... -v -short

# Run all integration tests across the entire project
test-integration-all:
	@echo "Running all integration tests across the project..."
	go test ./integration_tests/... -v

# Run ALL tests (unit + integration) across the entire project
test-all-project: test-unit-all test-integration-all
	@echo "All project tests completed."

# Run tests with verbose output showing individual test results
test-all-verbose:
	@echo "Running all tests with detailed output..."
	go test ./app/... ./integration_tests/... -v

# Legacy target for backward compatibility
test-all: test-all-project

# Legacy target for backward compatibility
test-integration: test-integration-all

# --- Test Count Targets ---
# Get test counts for different test types
test-count-unit:
	@echo "Counting unit tests..."
	@go test -list=. ./app/... | grep -c "^Test" || echo "0"

test-count-integration:
	@echo "Counting integration tests..."
	@go test -list=. ./integration_tests/... | grep -c "^Test" || echo "0"

test-count-all:
	@echo "=== TEST COUNT SUMMARY ==="
	@echo -n "Unit tests: "
	@go test -list=. ./app/... | grep -c "^Test" || echo "0"
	@echo -n "Integration tests: "
	@go test -list=. ./integration_tests/... | grep -c "^Test" || echo "0"
	@echo -n "Total tests: "
	@echo $$(( $$(go test -list=. ./app/... | grep -c "^Test" || echo "0") + $$(go test -list=. ./integration_tests/... | grep -c "^Test" || echo "0") ))

# --- Specific Integration Test Targets (Kept for individual runs if needed) ---
integration-leaderboard-service:
	@echo "Running leaderboard service integration tests..."
	go test ./integration_tests/modules/leaderboard/application_tests -v

integration-leaderboard-handlers:
	@echo "Running leaderboard handler integration tests..."
	go test ./integration_tests/modules/leaderboard/handler_tests -v

integration-user-service:
	@echo "Running user service integration tests..."
	go test ./integration_tests/modules/user/application_tests -v

integration-user-handlers:
	@echo "Running user handler integration tests..."
	go test ./integration_tests/modules/user/handler_tests -v

integration-score-service:
	@echo "Running score service integration tests..."
	go test ./integration_tests/modules/score/application_tests -v

integration-score-handlers:
	@echo "Running score handler integration tests..."
	go test ./integration_tests/modules/score/handler_tests -v

integration-round-service:
	@echo "Running round service integration tests..."
	go test ./integration_tests/modules/round/application_tests -v

integration-round-handlers:
	@echo "Running round handler integration tests..."
	go test ./integration_tests/modules/round/handler_tests -v

# --- Simplified Coverage Targets Using Go 1.20+ Native Coverage ---
REPORTS_DIR := ./reports

# Build instrumented binaries for coverage (keep this)
build-coverage:
	@echo "Building instrumented binaries for coverage..."
	-mkdir -p ./bin
	go build -cover -o ./bin/app-instrumented ./app
	go build -cover -o ./bin/bun-instrumented ./cmd/bun

# Enhanced coverage using both test and binary coverage
coverage-all: build-coverage
	@echo "Running all tests with coverage across entire project..."
	-mkdir -p $(REPORTS_DIR)
    # Set environment variable for binary coverage
	export GOCOVERDIR=$(REPORTS_DIR)/binary-coverage && \
	mkdir -p $$GOCOVERDIR && \
	go test -cover -coverprofile=$(REPORTS_DIR)/test-coverage.out ./app/... ./integration_tests/... && \
	go tool covdata textfmt -i=$$GOCOVERDIR -o=$(REPORTS_DIR)/binary-coverage.out 2>/dev/null || echo "No binary coverage data" && \
	if [ -f $(REPORTS_DIR)/binary-coverage.out ]; then \
  	go tool covdata merge -i=$(REPORTS_DIR) -o=$(REPORTS_DIR)/merged-coverage.out || cp $(REPORTS_DIR)/test-coverage.out $(REPORTS_DIR)/coverage.out; \
  	else \
  		cp $(REPORTS_DIR)/test-coverage.out $(REPORTS_DIR)/coverage.out; \
	fi
	@echo ""
	@echo "=========================================="
	@echo "OVERALL PROJECT COVERAGE SUMMARY:"
	@echo "=========================================="
	go tool cover -func $(REPORTS_DIR)/coverage.out
	@echo ""
	@echo "Total project coverage report generated: $(REPORTS_DIR)/coverage.out"

# Enhanced coverage with test counts using binary coverage
coverage-all-with-counts: build-coverage
	@echo "=== RUNNING ALL TESTS WITH COVERAGE ==="
	@echo ""
	@echo "=== TEST COUNT SUMMARY ==="
	@echo -n "Unit tests: "
	@go test -list=. ./app/... | grep -c "^Test" || echo "0"
	@echo -n "Integration tests: "
	@go test -list=. ./integration_tests/... | grep -c "^Test" || echo "0"
	@echo -n "Total tests: "
	@echo $$(( $$(go test -list=. ./app/... | grep -c "^Test" || echo "0") + $$(go test -list=. ./integration_tests/... | grep -c "^Test" || echo "0") ))
	@echo ""
	@echo "Running all tests with coverage across entire project..."
	-mkdir -p $(REPORTS_DIR)
	export GOCOVERDIR=$(REPORTS_DIR)/binary-coverage && \
	mkdir -p $$GOCOVERDIR && \
	go test -cover -coverprofile=$(REPORTS_DIR)/test-coverage.out ./app/... ./integration_tests/... -v && \
	go tool covdata textfmt -i=$$GOCOVERDIR -o=$(REPORTS_DIR)/binary-coverage.out 2>/dev/null || echo "No binary coverage data" && \
	if [ -f $(REPORTS_DIR)/binary-coverage.out ]; then \
  	go tool covdata merge -i=$(REPORTS_DIR) -o=$(REPORTS_DIR)/coverage.out || cp $(REPORTS_DIR)/test-coverage.out $(REPORTS_DIR)/coverage.out; \
	else \
  	cp $(REPORTS_DIR)/test-coverage.out $(REPORTS_DIR)/coverage.out; \
	fi
	@echo ""
	@echo "=========================================="
	@echo "OVERALL PROJECT COVERAGE SUMMARY:"
	@echo "=========================================="
	go tool cover -func $(REPORTS_DIR)/coverage.out
	@echo ""
	@echo "Total project coverage report generated: $(REPORTS_DIR)/coverage.out"

# Clean coverage reports (update to include binary coverage)
clean-coverage:
	@echo "Cleaning coverage reports..."
	-rm -rf $(REPORTS_DIR)
	-rm -rf ./bin/*-instrumented
# Generate HTML coverage report for entire project
coverage-html: coverage-all
	@echo "Generating HTML coverage report..."
	go tool cover -html $(REPORTS_DIR)/coverage.out -o $(REPORTS_DIR)/coverage.html
	@echo "HTML coverage report: $(REPORTS_DIR)/coverage.html"

# Run unit tests only with coverage
coverage-unit:
	@echo "Running unit tests with coverage..."
	-mkdir -p $(REPORTS_DIR)
	go test -cover -coverprofile=$(REPORTS_DIR)/unit-coverage.out -short ./app/...
	@echo ""
	@echo "=========================================="
	@echo "UNIT TEST COVERAGE SUMMARY:"
	@echo "=========================================="
	go tool cover -func $(REPORTS_DIR)/unit-coverage.out

# Run integration tests only with coverage  
coverage-integration:
	@echo "Running integration tests with coverage..."
	-mkdir -p $(REPORTS_DIR)
	go test -cover -coverprofile=$(REPORTS_DIR)/integration-coverage.out ./integration_tests/...
	@echo ""
	@echo "=========================================="
	@echo "INTEGRATION TEST COVERAGE SUMMARY:"
	@echo "=========================================="
	go tool cover -func $(REPORTS_DIR)/integration-coverage.out

# Clean coverage reports
clean-coverage:
	@echo "Cleaning coverage reports..."
	-rm -rf $(REPORTS_DIR)
	-rm -rf ./bin/*-instrumented

# --- Mock Generation Targets ---
MOCKGEN := mockgen
USER_DIR := ./app/modules/user
LB_DIR := ./app/modules/leaderboard
ROUND_DIR := ./app/modules/round
EVENTBUS_DIR := ./app/eventbus
SCORE_DIR := ./app/modules/score

mocks-user:
	$(MOCKGEN) -source=$(USER_DIR)/application/interface.go -destination=$(USER_DIR)/application/mocks/mock_service.go -package=mocks
	$(MOCKGEN) -source=$(USER_DIR)/infrastructure/handlers/interface.go -destination=$(USER_DIR)/infrastructure/handlers/mocks/mock_handlers.go -package=mocks
	$(MOCKGEN) -source=$(USER_DIR)/infrastructure/router/interface.go -destination=$(USER_DIR)/infrastructure/router/mocks/mock_router.go -package=mocks
	$(MOCKGEN) -source=$(USER_DIR)/infrastructure/repositories/interface.go -destination=$(USER_DIR)/infrastructure/repositories/mocks/mock_db.go -package=mocks

mocks-leaderboard:
	$(MOCKGEN) -source=$(LB_DIR)/application/interface.go -destination=$(LB_DIR)/application/mocks/mock_service.go -package=mocks
	$(MOCKGEN) -source=$(LB_DIR)/infrastructure/handlers/interface.go -destination=$(LB_DIR)/infrastructure/handlers/mocks/mock_handlers.go -package=mocks
	$(MOCKGEN) -source=$(LB_DIR)/infrastructure/router/interface.go -destination=$(LB_DIR)/infrastructure/router/mocks/mock_router.go -package=mocks
	$(MOCKGEN) -source=$(LB_DIR)/infrastructure/repositories/interface.go -destination=$(LB_DIR)/infrastructure/repositories/mocks/mock_db.go -package=mocks

mocks-round:
	$(MOCKGEN) -source=$(ROUND_DIR)/application/interface.go -destination=$(ROUND_DIR)/application/mocks/mock_service.go -package=mocks
	$(MOCKGEN) -source=$(ROUND_DIR)/infrastructure/handlers/interface.go -destination=$(ROUND_DIR)/infrastructure/handlers/mocks/mock_handlers.go -package=mocks
	$(MOCKGEN) -source=$(ROUND_DIR)/infrastructure/router/interface.go -destination=$(ROUND_DIR)/infrastructure/router/mocks/mock_router.go -package=mocks
	$(MOCKGEN) -source=$(ROUND_DIR)/infrastructure/repositories/interface.go -destination=$(ROUND_DIR)/infrastructure/repositories/mocks/mock_db.go -package=mocks
	$(MOCKGEN) -source=$(ROUND_DIR)/infrastructure/queue/service.go -destination=$(ROUND_DIR)/infrastructure/queue/mocks/mock_queue.go -package=mocks
	$(MOCKGEN) -source=$(ROUND_DIR)/utils/clock.go -destination=$(ROUND_DIR)/mocks/mock_clock.go -package=mocks
	$(MOCKGEN) -source=$(ROUND_DIR)/time_utils/time_conversion.go -destination=$(ROUND_DIR)/mocks/mock_conversion.go -package=mocks
	$(MOCKGEN) -source=$(ROUND_DIR)/utils/validator.go -destination=$(ROUND_DIR)/mocks/mock_validator.go -package=mocks

mocks-score:
	$(MOCKGEN) -source=$(SCORE_DIR)/application/interface.go -destination=$(SCORE_DIR)/application/mocks/mock_service.go -package=mocks
	$(MOCKGEN) -source=$(SCORE_DIR)/infrastructure/handlers/interface.go -destination=$(SCORE_DIR)/infrastructure/handlers/mocks/mock_handlers.go -package=mocks
	$(MOCKGEN) -source=$(SCORE_DIR)/infrastructure/router/interface.go -destination=$(SCORE_DIR)/infrastructure/router/mocks/mock_router.go -package=mocks
	$(MOCKGEN) -source=$(SCORE_DIR)/infrastructure/repositories/interface.go -destination=$(SCORE_DIR)/infrastructure/repositories/mocks/mock_db.go -package=mocks

mocks-eventbus:
	$(MOCKGEN) -source=../frolf-bot-shared/eventbus/eventbus.go -destination=$(EVENTBUS_DIR)/mocks/mock_eventbus.go -package=mocks

mocks-all: mocks-user mocks-eventbus mocks-leaderboard mocks-round mocks-score

build_version_ldflags := -X 'main.Version=$(shell git describe --tags --always)'

build-version:
	@echo "Building with version information..."
	go build -ldflags="$(build_version_ldflags)" ./...

# --- Database Configuration Helpers ---
# Show current database configuration
db-config:
	@echo "Database Configuration:"
	@echo "  URL: $(DB_URL)"
	@echo "  Host: $(DB_HOST)"
	@echo "  Port: $(DB_PORT)"
	@echo "  Database: $(DB_NAME)"
	@echo "  User: $(DB_USER)"
	@echo "  Password: [hidden]"

# Validate database connection
db-test:
	@echo "Testing database connection..."
	@PGPASSWORD="$(DB_PASS)" psql -h "$(DB_HOST)" -p "$(DB_PORT)" -U "$(DB_USER)" -d "$(DB_NAME)" -c "SELECT version();" || (echo "Database connection failed!" && exit 1)
	@echo "Database connection successful!"

# Set up environment for CI/CD (validates required variables)
ci-setup:
	@echo "Validating CI/CD environment..."
	@if [ -z "$(DATABASE_URL)" ]; then echo "ERROR: DATABASE_URL environment variable is required for CI/CD" && exit 1; fi
	@echo "DATABASE_URL is set"
	@echo "Testing database connection..."
	@$(MAKE) db-test
	@echo "CI/CD environment validation complete!"

# --- Help Target ---
help:
	@echo "Available targets:"
	@echo ""
	@echo "Database Migration:"
	@echo "  migrate-init          - Initialize database migrations"
	@echo "  migrate              - Run application migrations"
	@echo "  migrate-all          - Run River + application migrations"
	@echo "  rollback-all         - Rollback all migrations"
	@echo "  river-migrate-up     - Run River queue migrations"
	@echo "  river-migrate-down   - Rollback River migrations"
	@echo "  river-clean          - Clean up River tables and artifacts"
	@echo "  clean-all            - Clean all migrations and artifacts"
	@echo ""
	@echo "Database Configuration:"
	@echo "  db-config            - Show current database configuration"
	@echo "  db-test              - Test database connection"
	@echo "  ci-setup             - Validate environment for CI/CD"
	@echo ""
	@echo "Testing:"
	@echo "  test-unit-all        - Run all unit tests"
	@echo "  test-integration-all - Run all integration tests"
	@echo "  test-all-project     - Run all tests (unit + integration)"
	@echo "  test-count-all       - Show test counts"
	@echo ""
	@echo "Coverage:"
	@echo "  coverage-all         - Run tests with coverage"
	@echo "  coverage-html        - Generate HTML coverage report"
	@echo "  coverage-unit        - Unit test coverage only"
	@echo "  coverage-integration - Integration test coverage only"
	@echo ""
	@echo "Development:"
	@echo "  run                  - Run the application"
	@echo "  mocks-all            - Generate all mocks"
	@echo "  build-version        - Build with version info"
	@echo ""
	@echo "Environment Variables:"
	@echo "  DATABASE_URL         - Database connection string (required)"
	@echo "                         Can be set via environment or .env file"
	@echo "  Example: DATABASE_URL='postgres://user:pass@host:port/db'"
