.PHONY: migrate-init migrate migrate-all rollback-all run build clean
.PHONY: test-unit test-integration test-all test-count coverage-unit coverage-integration coverage-all coverage-html
.PHONY: test-user-unit test-user-integration test-user-all test-user-coverage
.PHONY: test-round-unit test-round-integration test-round-all test-round-coverage
.PHONY: test-leaderboard-unit test-leaderboard-integration test-leaderboard-all test-leaderboard-coverage
.PHONY: test-score-unit test-score-integration test-score-all test-score-coverage
.PHONY: mocks-all clean-coverage

# --- Configuration ---
REPORTS_DIR := ./reports
BIN_DIR := ./bin
MOCKGEN := mockgen

# Directories
USER_DIR := ./app/modules/user
LB_DIR := ./app/modules/leaderboard
ROUND_DIR := ./app/modules/round
SCORE_DIR := ./app/modules/score
EVENTBUS_DIR := ./app/eventbus

# Build version info
BUILD_VERSION := $(shell git describe --tags --always --dirty)
BUILD_LDFLAGS := -X 'main.Version=$(BUILD_VERSION)'

# Test configuration
TEST_TIMEOUT := 450s
TEST_PARALLEL := 1

# --- Database Migration ---
migrate-init:
	go run cmd/bun/main.go migrate init

migrate:
	go run cmd/bun/main.go migrate migrate

migrate-all: migrate-init migrate

rollback-all:
	go run cmd/bun/main.go migrate rollback

# --- Build Targets ---
build:
	@echo "Building application..."
	-mkdir -p $(BIN_DIR)
	go build -ldflags="$(BUILD_LDFLAGS)" -o $(BIN_DIR)/app .

run: build
	$(BIN_DIR)/app

# --- Project-Wide Test Targets ---
test-unit-fast:
	@echo "Running unit tests with optimized settings..."
	go test ./app/... -short -v -timeout=$(TEST_TIMEOUT) -parallel=$(TEST_PARALLEL) -tags=unit

test-integration-modules:
	@echo "Running integration tests by module in parallel..."
	@$(MAKE) -j4 test-user-integration test-round-integration test-leaderboard-integration test-score-integration

# Optimized coverage with module isolation
coverage-modules:
	@echo "Running modular coverage tests..."
	@$(MAKE) -j2 test-user-coverage test-round-coverage &
	@$(MAKE) -j2 test-leaderboard-coverage test-score-coverage &
	@wait

test-integration-parallel:
	@echo "Running integration tests in parallel by module..."
	@$(MAKE) -j2 test-user-integration test-round-integration &
	@$(MAKE) -j2 test-leaderboard-integration test-score-integration &
	@wait

test-unit:
	@echo "Running all unit tests..."
	go test ./app/... -short -v -timeout=$(TEST_TIMEOUT) -parallel=$(TEST_PARALLEL)

test-integration:
	@echo "Running all integration tests..."
	go test ./integration_tests/... -v -timeout=$(TEST_TIMEOUT) -parallel=2

test-all: test-unit test-integration

test-count:
	@echo "=== TEST COUNT SUMMARY ==="
	@echo -n "Unit test functions: "
	@go test -list=. ./app/... | grep -c "^Test" || echo "0"
	@echo -n "Integration test functions: "
	@go test -list=. ./integration_tests/... | grep -c "^Test" || echo "0"
	@echo -n "Total test functions: "
	@echo $$(( $$(go test -list=. ./app/... | grep -c "^Test" || echo "0") + $$(go test -list=. ./integration_tests/... | grep -c "^Test" || echo "0") ))

# --- Coverage Targets ---
coverage-unit:
	@echo "Running unit tests with coverage..."
	-mkdir -p $(REPORTS_DIR)
	go test -cover -coverprofile=$(REPORTS_DIR)/unit-coverage.out -short ./app/... -timeout=$(TEST_TIMEOUT) \
	-coverpkg=./app/... \
	| grep -v "mocks/" | grep -v "_test.go"
	@echo "Unit test coverage:"
	go tool cover -func $(REPORTS_DIR)/unit-coverage.out | tail -1

coverage-integration:
	@echo "Running integration tests with coverage..."
	-mkdir -p $(REPORTS_DIR)
	go test -cover -coverprofile=$(REPORTS_DIR)/integration-coverage.out ./integration_tests/... -timeout=$(TEST_TIMEOUT) -parallel=2 \
	-coverpkg=./app/... \
	| grep -v "mocks/" | grep -v "_test.go"
	@echo "Integration test coverage:"
	go tool cover -func $(REPORTS_DIR)/integration-coverage.out | tail -1

# Replace the coverage-combined target with proper tab formatting:

coverage-combined:
	@echo "Running combined unit and integration tests with coverage..."
	-mkdir -p $(REPORTS_DIR)
	@$(MAKE) test-count
	@echo ""
	@echo "Running unit tests with coverage..."
	@go test -coverprofile=$(REPORTS_DIR)/unit-coverage.out -short ./app/... -timeout=$(TEST_TIMEOUT) -coverpkg=./app/... -v 2>&1 | tee $(REPORTS_DIR)/unit-test-output.log; \
	UNIT_EXIT_CODE=$$?; \
	echo "Unit tests completed with exit code: $$UNIT_EXIT_CODE"
	@echo ""
	@echo "Running integration tests with coverage..."
	@go test -coverprofile=$(REPORTS_DIR)/integration-coverage.out ./integration_tests/... -timeout=$(TEST_TIMEOUT) -parallel=1 -coverpkg=./app/... -v 2>&1 | tee $(REPORTS_DIR)/integration-test-output.log; \
	INTEGRATION_EXIT_CODE=$$?; \
	echo "Integration tests completed with exit code: $$INTEGRATION_EXIT_CODE"
	@echo ""
	@echo "Validating coverage files..."
	@if [ -f $(REPORTS_DIR)/unit-coverage.out ]; then \
			echo "Unit coverage file size: $$(wc -l < $(REPORTS_DIR)/unit-coverage.out) lines"; \
			head -1 $(REPORTS_DIR)/unit-coverage.out; \
	else \
			echo "No unit coverage file generated"; \
	fi
	@if [ -f $(REPORTS_DIR)/integration-coverage.out ]; then \
			echo "Integration coverage file size: $$(wc -l < $(REPORTS_DIR)/integration-coverage.out) lines"; \
			head -1 $(REPORTS_DIR)/integration-coverage.out; \
	else \
			echo "No integration coverage file generated"; \
	fi
	@echo ""
	@echo "Merging coverage profiles..."
	@if [ -f $(REPORTS_DIR)/unit-coverage.out ] && [ -s $(REPORTS_DIR)/unit-coverage.out ]; then \
		if [ -f $(REPORTS_DIR)/integration-coverage.out ] && [ -s $(REPORTS_DIR)/integration-coverage.out ]; then \
			echo "mode: set" > $(REPORTS_DIR)/combined-coverage.out; \
			tail -n +2 $(REPORTS_DIR)/unit-coverage.out >> $(REPORTS_DIR)/combined-coverage.out; \
			tail -n +2 $(REPORTS_DIR)/integration-coverage.out >> $(REPORTS_DIR)/combined-coverage.out; \
			echo "Combined unit and integration coverage"; \
		else \
			cp $(REPORTS_DIR)/unit-coverage.out $(REPORTS_DIR)/combined-coverage.out; \
			echo "Using unit coverage only"; \
		fi; \
	elif [ -f $(REPORTS_DIR)/integration-coverage.out ] && [ -s $(REPORTS_DIR)/integration-coverage.out ]; then \
		cp $(REPORTS_DIR)/integration-coverage.out $(REPORTS_DIR)/combined-coverage.out; \
		echo "Using integration coverage only"; \
	else \
		echo "No coverage data available"; \
		echo "mode: set" > $(REPORTS_DIR)/combined-coverage.out; \
	fi
	@echo ""
	@echo "=========================================="
	@echo "TEST EXECUTION SUMMARY:"
	@echo "=========================================="
	@echo -n "Unit test cases: "
	@grep -c "=== RUN" $(REPORTS_DIR)/unit-test-output.log 2>/dev/null || echo "Could not determine"
	@echo -n "Unit tests passed: "
	@grep -c "--- PASS:" $(REPORTS_DIR)/unit-test-output.log 2>/dev/null || echo "0"
	@echo -n "Unit tests failed: "
	@grep -c "--- FAIL:" $(REPORTS_DIR)/unit-test-output.log 2>/dev/null || echo "0"
	@echo -n "Integration test cases: "
	@grep -c "=== RUN" $(REPORTS_DIR)/integration-test-output.log 2>/dev/null || echo "Could not determine"
	@echo -n "Integration tests passed: "
	@grep -c "--- PASS:" $(REPORTS_DIR)/integration-test-output.log 2>/dev/null || echo "0"
	@echo -n "Integration tests failed: "
	@grep -c "--- FAIL:" $(REPORTS_DIR)/integration-test-output.log 2>/dev/null || echo "0"
	@echo ""
	@echo "=========================================="
	@echo "COMBINED COVERAGE SUMMARY:"
	@echo "=========================================="
	@if [ -f $(REPORTS_DIR)/combined-coverage.out ] && [ -s $(REPORTS_DIR)/combined-coverage.out ]; then \
		echo "Total combined coverage:"; \
		go tool cover -func $(REPORTS_DIR)/combined-coverage.out | tail -1; \
		echo ""; \
		echo "Business Logic Coverage by Module:"; \
		go tool cover -func $(REPORTS_DIR)/combined-coverage.out | grep -E "app/modules/.*/application/" | grep -v "mocks/mock_" | sort || echo "No application logic coverage found"; \
	else \
		echo "No coverage data available for analysis"; \
	fi

# Alternative: Pure build-time coverage for integration testing
coverage-integration-builtin:
	@echo "Running integration tests with pure build-time coverage..."
	-mkdir -p $(REPORTS_DIR)
	-mkdir -p $(REPORTS_DIR)/integration-coverage-data
	@echo "Building coverage-instrumented binary..."
	go build -cover -o $(BIN_DIR)/app-coverage .
	@echo "Running integration tests with coverage instrumentation..."
	GOCOVERDIR=$(REPORTS_DIR)/integration-coverage-data go test ./integration_tests/... -v -timeout=$(TEST_TIMEOUT) -parallel=1
	@echo "Coverage report from build-time instrumentation:"
	go tool covdata percent -i=$(REPORTS_DIR)/integration-coverage-data
	@echo ""
	@echo "Converting to profile format for detailed analysis..."
	go tool covdata textfmt -i=$(REPORTS_DIR)/integration-coverage-data -o $(REPORTS_DIR)/integration-builtin-coverage.out
	@echo "Detailed coverage:"
	@go tool cover -func $(REPORTS_DIR)/integration-builtin-coverage.out | grep -E "app/modules/.*/application/" | sort

# Quick combined coverage without detailed output
coverage-combined-quick:
	@echo "Running quick combined coverage test..."
	-mkdir -p $(REPORTS_DIR)
	go test -cover -coverprofile=$(REPORTS_DIR)/quick-combined-coverage.out \
	-coverpkg=./app/... \
	-timeout=$(TEST_TIMEOUT) \
	-parallel=4 \
	./app/... ./integration_tests/... > /dev/null 2>&1
	@go run cmd/coverage-filter/main.go $(REPORTS_DIR)/quick-combined-coverage.out $(REPORTS_DIR)/quick-combined-coverage-filtered.out
	@echo "Quick combined coverage:"
	@go tool cover -func $(REPORTS_DIR)/quick-combined-coverage-filtered.out | tail -1

# --- User Module Tests ---
test-user-unit:
	@echo "Running user module unit tests..."
	go test $(USER_DIR)/... -short -v -timeout=$(TEST_TIMEOUT)

test-user-integration:
	@echo "Running user module integration tests..."
	go test ./integration_tests/modules/user/... -v -timeout=$(TEST_TIMEOUT)

test-user-all: test-user-unit test-user-integration

test-user-coverage:
	@echo "Running user module tests with coverage..."
	-mkdir -p $(REPORTS_DIR)
	go test -cover -coverprofile=$(REPORTS_DIR)/user-coverage.out $(USER_DIR)/... ./integration_tests/modules/user/... -timeout=$(TEST_TIMEOUT) \
	-coverpkg=$(USER_DIR)/... \
	-v
	@echo "Post-processing user module coverage..."
	@go run cmd/coverage-filter/main.go $(REPORTS_DIR)/user-coverage.out $(REPORTS_DIR)/user-coverage-filtered.out
	@echo "User module coverage:"
	go tool cover -func $(REPORTS_DIR)/user-coverage-filtered.out | tail -1

# --- Round Module Tests ---
test-round-unit:
	@echo "Running round module unit tests..."
	go test $(ROUND_DIR)/... -short -v -timeout=$(TEST_TIMEOUT)

test-round-integration:
	@echo "Running round module integration tests..."
	go test ./integration_tests/modules/round/... -v -timeout=$(TEST_TIMEOUT)

test-round-all: test-round-unit test-round-integration

test-round-coverage:
	@echo "Running round module tests with coverage..."
	-mkdir -p $(REPORTS_DIR)
	go test -cover -coverprofile=$(REPORTS_DIR)/round-coverage.out $(ROUND_DIR)/... ./integration_tests/modules/round/... -timeout=$(TEST_TIMEOUT) \
	-coverpkg=$(ROUND_DIR)/... \
	-v
	@echo "Post-processing round module coverage..."
	@go run cmd/coverage-filter/main.go $(REPORTS_DIR)/round-coverage.out $(REPORTS_DIR)/round-coverage-filtered.out
	@echo "Round module coverage:"
	go tool cover -func $(REPORTS_DIR)/round-coverage-filtered.out | tail -1

# --- Leaderboard Module Tests ---
test-leaderboard-unit:
	@echo "Running leaderboard module unit tests..."
	go test $(LB_DIR)/... -short -v -timeout=$(TEST_TIMEOUT)

test-leaderboard-integration:
	@echo "Running leaderboard module integration tests..."
	go test ./integration_tests/modules/leaderboard/... -v -timeout=$(TEST_TIMEOUT)

test-leaderboard-all: test-leaderboard-unit test-leaderboard-integration

test-leaderboard-coverage:
	@echo "Running leaderboard module tests with coverage..."
	-mkdir -p $(REPORTS_DIR)
	go test -cover -coverprofile=$(REPORTS_DIR)/leaderboard-coverage.out $(LB_DIR)/... ./integration_tests/modules/leaderboard/... -timeout=$(TEST_TIMEOUT) \
	-coverpkg=$(LB_DIR)/... \
	-v
	@echo "Post-processing leaderboard module coverage..."
	@go run cmd/coverage-filter/main.go $(REPORTS_DIR)/leaderboard-coverage.out $(REPORTS_DIR)/leaderboard-coverage-filtered.out
	@echo "Leaderboard module coverage:"
	go tool cover -func $(REPORTS_DIR)/leaderboard-coverage-filtered.out | tail -1

# --- Score Module Tests ---
test-score-unit:
	@echo "Running score module unit tests..."
	go test $(SCORE_DIR)/... -short -v -timeout=$(TEST_TIMEOUT)

test-score-integration:
	@echo "Running score module integration tests..."
	go test ./integration_tests/modules/score/... -v -timeout=$(TEST_TIMEOUT)

test-score-all: test-score-unit test-score-integration

test-score-coverage:
	@echo "Running score module tests with coverage..."
	-mkdir -p $(REPORTS_DIR)
	go test -cover -coverprofile=$(REPORTS_DIR)/score-coverage.out $(SCORE_DIR)/... ./integration_tests/modules/score/... -timeout=$(TEST_TIMEOUT) \
	-coverpkg=$(SCORE_DIR)/... \
	-v
	@echo "Post-processing score module coverage..."
	@go run cmd/coverage-filter/main.go $(REPORTS_DIR)/score-coverage.out $(REPORTS_DIR)/score-coverage-filtered.out
	@echo "Score module coverage:"
	go tool cover -func $(REPORTS_DIR)/score-coverage-filtered.out | tail -1

# --- Mock Generation ---
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

mocks-all: mocks-user mocks-leaderboard mocks-round mocks-score mocks-eventbus

# --- Cleanup ---
clean-coverage:
	@echo "Cleaning coverage reports..."
	-rm -rf $(REPORTS_DIR)

clean: clean-coverage
	@echo "Cleaning build artifacts..."
	-rm -rf $(BIN_DIR)
	-rm -rf tmp/

# --- Help ---
help:
	@echo "Available targets:"
	@echo ""
	@echo "Application:"
	@echo "  run                      - Build and run the application"
	@echo "  build                    - Build the application"
	@echo "  migrate-all              - Run database migrations"
	@echo ""
	@echo "Project-wide testing:"
	@echo "  test-all                 - Run all tests"
	@echo "  test-unit                - Run all unit tests"
	@echo "  test-integration         - Run all integration tests"
	@echo "  coverage-all             - Run all tests with coverage (recommended)"
	@echo "  coverage-html            - Generate HTML coverage report"
	@echo ""
	@echo "Module-specific testing:"
	@echo "  test-user-unit           - Run user module unit tests"
	@echo "  test-user-integration    - Run user module integration tests"
	@echo "  test-user-all            - Run all user module tests"
	@echo "  test-user-coverage       - Run user module tests with coverage"
	@echo ""
	@echo "  test-round-unit          - Run round module unit tests"
	@echo "  test-round-integration   - Run round module integration tests"
	@echo "  test-round-all           - Run all round module tests"
	@echo "  test-round-coverage      - Run round module tests with coverage"
	@echo ""
	@echo "  test-leaderboard-unit    - Run leaderboard module unit tests"
	@echo "  test-leaderboard-integration - Run leaderboard module integration tests"
	@echo "  test-leaderboard-all     - Run all leaderboard module tests"
	@echo "  test-leaderboard-coverage - Run leaderboard module tests with coverage"
	@echo ""
	@echo "  test-score-unit          - Run score module unit tests"
	@echo "  test-score-integration   - Run score module integration tests"
	@echo "  test-score-all           - Run all score module tests"
	@echo "  test-score-coverage      - Run score module tests with coverage"
	@echo ""
	@echo "Utilities:"
	@echo "  test-count               - Show test count summary"
	@echo "  mocks-all                - Generate all mocks"
	@echo "  clean                    - Clean build artifacts and reports"
