.PHONY: migrate-init migrate migrate-all rollback-all run
.PHONY: test-unit-all test-integration-all test-all-project test-all-verbose
.PHONY: test-with-summary test-unit-summary test-integration-summary
.PHONY: test-quick test-silent test-json test-module
.PHONY: test-integration-module test-integration-user test-integration-round test-integration-score test-integration-leaderboard
.PHONY: test-integration-round-summary test-round-summary
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

# Run tests with failure summary at the end
test-with-summary:
	@echo "Running all tests with failure summary..."
	@TEMP_FILE=$$(mktemp) && \
	(go test ./app/... ./integration_tests/... -v 2>&1 | tee $$TEMP_FILE; \
	EXIT_CODE=$${PIPESTATUS[0]}; \
	echo ""; \
	echo "=========================================="; \
	echo "TEST SUMMARY"; \
	echo "=========================================="; \
	if [ $$EXIT_CODE -eq 0 ]; then \
		echo "✅ ALL TESTS PASSED"; \
		TOTAL_PASSED=$$(grep -c "^--- PASS:" $$TEMP_FILE || echo "0"); \
		echo "Total passed: $$TOTAL_PASSED"; \
	else \
		echo "❌ SOME TESTS FAILED"; \
		echo ""; \
		TOTAL_PASSED=$$(grep -c "^--- PASS:" $$TEMP_FILE || echo "0"); \
		TOTAL_FAILED=$$(grep -c "^--- FAIL:" $$TEMP_FILE || echo "0"); \
		echo "📊 Test Results: $$TOTAL_PASSED passed, $$TOTAL_FAILED failed"; \
		echo ""; \
		echo "🔍 FAILED TEST PACKAGES:"; \
		grep "^FAIL[[:space:]]" $$TEMP_FILE | sed 's/^FAIL[[:space:]]*/  • /' || echo "No package failures found"; \
		echo ""; \
		echo "❌ INDIVIDUAL FAILED TESTS:"; \
		grep "^--- FAIL:" $$TEMP_FILE | sed 's/^--- FAIL: /  • /' | head -20 || echo "No individual test failures found"; \
		if [ $$(grep -c "^--- FAIL:" $$TEMP_FILE || echo "0") -gt 20 ]; then \
			echo "  ... and more (showing first 20 failures)"; \
		fi; \
		echo ""; \
		echo "📋 FAILURE DETAILS (first few):"; \
		grep -A 5 -B 1 "^--- FAIL:" $$TEMP_FILE | head -30 | grep -v "^--$$" || echo "No detailed failure info found"; \
	fi; \
	rm -f $$TEMP_FILE; \
	exit $$EXIT_CODE)

# Run unit tests only with failure summary
test-unit-summary:
	@echo "Running unit tests with failure summary..."
	@TEMP_FILE=$$(mktemp) && \
	(go test ./app/... -v -short 2>&1 | tee $$TEMP_FILE; \
	EXIT_CODE=$${PIPESTATUS[0]}; \
	echo ""; \
	echo "=========================================="; \
	echo "UNIT TEST SUMMARY"; \
	echo "=========================================="; \
	if [ $$EXIT_CODE -eq 0 ]; then \
		echo "✅ ALL UNIT TESTS PASSED"; \
		grep "^--- PASS:" $$TEMP_FILE | wc -l | xargs printf "Total passed: %s\n"; \
	else \
		echo "❌ SOME UNIT TESTS FAILED"; \
		echo ""; \
		echo "FAILED TESTS:"; \
		grep -E "^--- FAIL:|^FAIL" $$TEMP_FILE || echo "No specific test failures found"; \
		echo ""; \
		echo "FAILURE DETAILS:"; \
		grep -A 10 -B 2 "FAIL\|panic:" $$TEMP_FILE | grep -v "^--$$" || echo "No detailed failure info found"; \
	fi; \
	rm -f $$TEMP_FILE; \
	exit $$EXIT_CODE)

# Run integration tests with failure summary
test-integration-summary:
	@echo "Running integration tests with failure summary..."
	@TEMP_FILE=$$(mktemp) && \
	(go test ./integration_tests/... -v 2>&1 | tee $$TEMP_FILE; \
	EXIT_CODE=$${PIPESTATUS[0]}; \
	echo ""; \
	echo "=========================================="; \
	echo "INTEGRATION TEST SUMMARY"; \
	echo "=========================================="; \
	if [ $$EXIT_CODE -eq 0 ]; then \
		echo "✅ ALL INTEGRATION TESTS PASSED"; \
		grep "^--- PASS:" $$TEMP_FILE | wc -l | xargs printf "Total passed: %s\n"; \
	else \
		echo "❌ SOME INTEGRATION TESTS FAILED"; \
		echo ""; \
		echo "FAILED TESTS:"; \
		grep -E "^--- FAIL:|^FAIL" $$TEMP_FILE || echo "No specific test failures found"; \
		echo ""; \
		echo "FAILURE DETAILS:"; \
		grep -A 10 -B 2 "FAIL\|panic:" $$TEMP_FILE | grep -v "^--$$" || echo "No detailed failure info found"; \
	fi; \
	rm -f $$TEMP_FILE; \
	exit $$EXIT_CODE)

# Quick unit test check (fast feedback loop)
test-quick:
	@echo "Running quick unit tests (no integration tests)..."
	@TEMP_FILE=$$(mktemp) && \
	(go test ./app/... -short 2>&1 | tee $$TEMP_FILE; \
	EXIT_CODE=$${PIPESTATUS[0]}; \
	if [ $$EXIT_CODE -ne 0 ]; then \
		echo ""; \
		echo "❌ UNIT TEST FAILURES:"; \
		grep -E "^--- FAIL:|^FAIL" $$TEMP_FILE || echo "No specific test failures found"; \
	else \
		echo "✅ All unit tests passed!"; \
	fi; \
	rm -f $$TEMP_FILE; \
	exit $$EXIT_CODE)

# Silent test run - only shows results, no progress
test-silent:
	@echo "Running tests silently..."
	@TEMP_FILE=$$(mktemp) && \
	(go test ./app/... ./integration_tests/... 2>&1 > $$TEMP_FILE; \
	EXIT_CODE=$$?; \
	if [ $$EXIT_CODE -eq 0 ]; then \
		echo "✅ ALL TESTS PASSED"; \
	else \
		echo "❌ TEST FAILURES DETECTED:"; \
		grep -E "^--- FAIL:|^FAIL" $$TEMP_FILE || echo "Check full output for details"; \
	fi; \
	rm -f $$TEMP_FILE; \
	exit $$EXIT_CODE)

# Test with JSON output for parsing by tools/CI
test-json:
	@echo "Running tests with JSON output..."
	go test ./app/... ./integration_tests/... -json

# Test specific module with summary
test-module:
	@if [ -z "$(MODULE)" ]; then \
		echo "Usage: make test-module MODULE=user|round|score|leaderboard"; \
		echo "Example: make test-module MODULE=round"; \
		exit 1; \
	fi
	@echo "Running tests for $(MODULE) module with summary..."
	@TEMP_FILE=$$(mktemp) && \
	(go test ./app/modules/$(MODULE)/... ./integration_tests/modules/$(MODULE)/... -v 2>&1 | tee $$TEMP_FILE; \
	EXIT_CODE=$${PIPESTATUS[0]}; \
	echo ""; \
	echo "=========================================="; \
	echo "$(MODULE) MODULE TEST SUMMARY"; \
	echo "=========================================="; \
	if [ $$EXIT_CODE -eq 0 ]; then \
		echo "✅ ALL $(MODULE) TESTS PASSED"; \
		grep "^--- PASS:" $$TEMP_FILE | wc -l | xargs printf "Total passed: %s\n"; \
	else \
		echo "❌ SOME $(MODULE) TESTS FAILED"; \
		echo ""; \
		echo "FAILED TESTS:"; \
		grep -E "^--- FAIL:|^FAIL" $$TEMP_FILE || echo "No specific test failures found"; \
	fi; \
	rm -f $$TEMP_FILE; \
	exit $$EXIT_CODE)

# Run only integration tests for a specific module with a summary
test-integration-module:
	@if [ -z "$(MODULE)" ]; then \
		echo "Usage: make test-integration-module MODULE=user|round|score|leaderboard"; \
		echo "Example: make test-integration-module MODULE=round"; \
		exit 1; \
	fi
	@echo "Running integration tests for $(MODULE) with summary..."
	@TEMP_FILE=$$(mktemp) && \
	(go test ./integration_tests/modules/$(MODULE)/... -v 2>&1 | tee $$TEMP_FILE; \
	EXIT_CODE=$${PIPESTATUS[0]}; \
	echo ""; \
	echo "=========================================="; \
	echo "INTEGRATION TEST SUMMARY: $(MODULE)"; \
	echo "=========================================="; \
	if [ $$EXIT_CODE -eq 0 ]; then \
		echo "✅ ALL INTEGRATION TESTS PASSED"; \
		grep "^--- PASS:" $$TEMP_FILE | wc -l | xargs printf "Total passed: %s\n"; \
	else \
		echo "❌ SOME INTEGRATION TESTS FAILED"; \
		echo ""; \
		echo "FAILED TESTS:"; \
		grep -E "^--- FAIL:|^FAIL" $$TEMP_FILE || echo "No specific test failures found"; \
		echo ""; \
		echo "FAILURE DETAILS:"; \
		grep -A 10 -B 2 "FAIL\|panic:" $$TEMP_FILE | grep -v "^--$$" || echo "No detailed failure info found"; \
	fi; \
	rm -f $$TEMP_FILE; \
	exit $$EXIT_CODE)

# Convenience module-specific integration targets
test-integration-user: ; $(MAKE) test-integration-module MODULE=user
test-integration-round: ; $(MAKE) test-integration-module MODULE=round
test-integration-score: ; $(MAKE) test-integration-module MODULE=score
test-integration-leaderboard: ; $(MAKE) test-integration-module MODULE=leaderboard

# Round module specific test targets with summary
test-integration-round-summary:
	@echo "Running round integration tests with failure summary..."
	@TEMP_FILE=$$(mktemp) && \
	(go test ./integration_tests/modules/round/... -v 2>&1 | tee $$TEMP_FILE; \
	EXIT_CODE=$${PIPESTATUS[0]}; \
	echo ""; \
	echo "=========================================="; \
	echo "ROUND INTEGRATION TEST SUMMARY"; \
	echo "=========================================="; \
	if [ $$EXIT_CODE -eq 0 ]; then \
		echo "✅ ALL ROUND INTEGRATION TESTS PASSED"; \
		grep "^--- PASS:" $$TEMP_FILE | wc -l | xargs printf "Total passed: %s\n"; \
	else \
		echo "❌ SOME ROUND INTEGRATION TESTS FAILED"; \
		echo ""; \
		echo "FAILED TESTS:"; \
		grep -E "^--- FAIL:|^FAIL" $$TEMP_FILE || echo "No specific test failures found"; \
		echo ""; \
		echo "FAILURE DETAILS:"; \
		grep -A 10 -B 2 "FAIL\|panic:" $$TEMP_FILE | grep -v "^--$$" || echo "No detailed failure info found"; \
	fi; \
	rm -f $$TEMP_FILE; \
	exit $$EXIT_CODE)

test-round-summary:
	@echo "Running all round module tests with failure summary..."
	@TEMP_FILE=$$(mktemp) && \
	(go test ./app/modules/round/... ./integration_tests/modules/round/... -v 2>&1 | tee $$TEMP_FILE; \
	EXIT_CODE=$${PIPESTATUS[0]}; \
	echo ""; \
	echo "=========================================="; \
	echo "ROUND MODULE TEST SUMMARY"; \
	echo "=========================================="; \
	if [ $$EXIT_CODE -eq 0 ]; then \
		echo "✅ ALL ROUND MODULE TESTS PASSED"; \
		grep "^--- PASS:" $$TEMP_FILE | wc -l | xargs printf "Total passed: %s\n"; \
	else \
		echo "❌ SOME ROUND MODULE TESTS FAILED"; \
		echo ""; \
		echo "FAILED TESTS:"; \
		grep -E "^--- FAIL:|^FAIL" $$TEMP_FILE || echo "No specific test failures found"; \
		echo ""; \
		echo "FAILURE DETAILS:"; \
		grep -A 10 -B 2 "FAIL\|panic:" $$TEMP_FILE | grep -v "^--$$" || echo "No detailed failure info found"; \
	fi; \
	rm -f $$TEMP_FILE; \
	exit $$EXIT_CODE)

# -------------------------------
# Coverage (Sane & Actionable)
# -------------------------------

REPORTS_DIR := reports

# Only business logic packages
COVER_PKGS := $(shell go list ./app/... | \
	grep -E '/application|/infrastructure/handlers' | \
	grep -vE '/mocks|/migrations' | \
	tr '\n' ',' | sed 's/,$$//')

coverage:
	@mkdir -p $(REPORTS_DIR)
	@echo "=========================================="
	@echo "Running unit + integration tests"
	@echo "Measured packages:"
	@echo "  $(COVER_PKGS)"
	@echo "=========================================="

	@go test -v \
		./app/... \
		./integration_tests/... \
		-coverpkg=$(COVER_PKGS) \
		-coverprofile=$(REPORTS_DIR)/coverage.out

	@echo "=========================================="
	@echo "COVERAGE TOTAL:"
	@go tool cover -func=$(REPORTS_DIR)/coverage.out | grep total
	@echo "=========================================="
	@echo "HTML REPORT:"
	@echo "go tool cover -html=$(REPORTS_DIR)/coverage.out"

REPORTS_DIR := reports

UNIT_COVER_PKGS := $(shell go list ./app/... | \
	grep -E '/application|/shared|/infrastructure/handlers' | \
	grep -vE '/mocks|/migrations|/router|/models|/interfaces' | \
	tr '\n' ',' | sed 's/,$$//')

coverage-unit:
	@mkdir -p $(REPORTS_DIR)
	@echo "=========================================="
	@echo "UNIT / API COVERAGE"
	@echo "Measured packages:"
	@echo "  $(UNIT_COVER_PKGS)"
	@echo "=========================================="

	@go test -v ./app/... \
		-covermode=atomic \
		-coverpkg=$(UNIT_COVER_PKGS) \
		-coverprofile=$(REPORTS_DIR)/coverage-unit.out

	@echo "=========================================="
	@echo "UNIT COVERAGE TOTAL:"
	@go tool cover -func=$(REPORTS_DIR)/coverage-unit.out | grep total
	@echo "=========================================="
	@echo "HTML REPORT:"
	@echo "go tool cover -html=$(REPORTS_DIR)/coverage-unit.out"

INTEGRATION_COVER_PKGS := $(shell go list ./app/... | \
	grep -E '/repositories|/events' | \
	grep -vE '/migrations|/mocks' | \
	tr '\n' ',' | sed 's/,$$//')

coverage-integration:
	@mkdir -p $(REPORTS_DIR)
	@echo "=========================================="
	@echo "INTEGRATION COVERAGE"
	@echo "Measured packages:"
	@echo "  $(INTEGRATION_COVER_PKGS)"
	@echo "=========================================="

	@go test -v ./integration_tests/... \
		-covermode=atomic \
		-coverpkg=$(INTEGRATION_COVER_PKGS) \
		-coverprofile=$(REPORTS_DIR)/coverage-integration.out

	@echo "=========================================="
	@echo "INTEGRATION COVERAGE TOTAL:"
	@go tool cover -func=$(REPORTS_DIR)/coverage-integration.out | grep total
	@echo "=========================================="


# Enhanced coverage with test counts
coverage-all-with-counts:
	@echo "=== RUNNING ALL TESTS WITH COVERAGE ==="
	@echo ""
	@echo "=== TEST COUNT SUMMARY ==="
	@echo -n "Unit tests: "
	@go test -list=. ./app/... 2>/dev/null | grep -c "^Test" || echo "0"
	@echo -n "Integration tests: "
	@go test -list=. ./integration_tests/... 2>/dev/null | grep -c "^Test" || echo "0"
	@echo -n "Total tests: "
	@echo $$(( $$(go test -list=. ./app/... 2>/dev/null | grep -c "^Test" || echo "0") + $$(go test -list=. ./integration_tests/... 2>/dev/null | grep -c "^Test" || echo "0") ))
	@echo ""
	@$(MAKE) coverage-all

# Generate HTML coverage report
coverage-html: coverage-all
	@echo "Generating HTML coverage report..."
	@go tool cover -html=$(COV_TEXT) -o $(REPORTS_DIR)/coverage.html
	@echo "HTML report: $(REPORTS_DIR)/coverage.html"
	@echo "Open with: open $(REPORTS_DIR)/coverage.html"

# Clean coverage reports
clean-coverage:
	@echo "Cleaning coverage reports..."
	-rm -rf $(REPORTS_DIR)
	-rm -rf ./bin/*-instrumented

# --- Mock Generation Targets ---
# No more mocks - using Fakes
# See modules/leaderboard/application/fake_test.go, etc.

mocks-all:
	@echo "Mocks no longer used - using Fakes."

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
	@echo "  test-with-summary    - Run all tests with failure summary"
	@echo "  test-unit-summary    - Run unit tests with failure summary"
	@echo "  test-integration-summary - Run integration tests with failure summary"
	@echo "  test-quick           - Quick unit tests only (fast feedback)"
	@echo "  test-silent          - Run tests silently (results only)"
	@echo "  test-json            - Run tests with JSON output"
	@echo "  test-module MODULE=x - Test specific module (user|round|score|leaderboard)"
	@echo "  test-integration-module MODULE=x - Run integration tests for a specific module (user|round|score|leaderboard)"
	@echo "  test-integration-user|round|score|leaderboard - Convenience targets for integration tests per module"
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
