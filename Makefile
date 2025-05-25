.PHONY: migrate-init migrate migrate-all rollback-all run
.PHONY: test-all test-integration integration-leaderboard-service integration-leaderboard-handlers
.PHONY: integration-user-service integration-user-handlers integration-score-service integration-score-handlers
.PHONY: integration-round-service integration-round-handlers
.PHONY: coverage-unit-leaderboard-handlers coverage-integration-leaderboard-handlers # Keep existing handler targets, but update their output
.PHONY: coverage-unit-leaderboard-application coverage-integration-leaderboard-application # Added new application coverage targets
.PHONY: coverage-leaderboard-module-combined coverage-leaderboard-module-html clean-coverage # Added new combined module targets and clean
.PHONY: coverage-unit-round-handlers coverage-integration-round-handlers # New round handler coverage targets
.PHONY: coverage-unit-round-application coverage-integration-round-application # New round application coverage targets
.PHONY: coverage-round-module-combined coverage-round-module-html # New round combined module targets
.PHONY: mocks-user mocks-leaderboard mocks-round mocks-score mocks-eventbus mocks-all build-version

# --- Database Migration and Run Targets ---
migrate-init:
	go run cmd/bun/main.go migrate init

migrate:
	go run cmd/bun/main.go migrate migrate

migrate-all: migrate-init migrate

rollback-all:
	go run cmd/bun/main.go migrate rollback

run:
	go run cmd/app/main.go

# --- General Test Targets ---
# Run all tests from the root
test-all:
	@echo "Running all tests from root..."
	go test ./... -v

# Run only integration tests from the root
test-integration:
	@echo "Running integration tests from root..."
	go test ./integration_tests/... -v

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


# --- Coverage Targets for Leaderboard Module ---

# Define package paths for clarity
LEADERBOARD_APPLICATION_UNIT_TEST_PKG := ./app/modules/leaderboard/application
LEADERBOARD_APPLICATION_INTEGRATION_TEST_PKG := ./integration_tests/modules/leaderboard/application_tests
LEADERBOARD_HANDLER_UNIT_TEST_PKG := ./app/modules/leaderboard/infrastructure/handlers
LEADERBOARD_HANDLER_INTEGRATION_TEST_PKG := ./integration_tests/modules/leaderboard/handler_tests

# Define packages to measure coverage for
# Unit tests should measure coverage of the package being tested.
LEADERBOARD_APPLICATION_COVERAGE_PKGS := ./app/modules/leaderboard/application
LEADERBOARD_HANDLER_COVERAGE_PKGS := ./app/modules/leaderboard/infrastructure/handlers
# Integration tests and combined report should measure coverage for relevant packages in the module.
# Adjust this list if you want to include more packages from the leaderboard module.
LEADERBOARD_MODULE_COVERAGE_PKGS := ./app/modules/leaderboard/application,./app/modules/leaderboard/infrastructure/handlers,./app/modules/leaderboard/infrastructure/repositories

# Define directories and filenames for coverage data using text profile format
TEMP_COVERAGE_DATA_ROOT_MODULE := .coverage_temp/leaderboard_module
UNIT_HANDLER_COV_FILE := $(TEMP_COVERAGE_DATA_ROOT_MODULE)/unit_handler.cov
INTEGRATION_HANDLER_COV_FILE := $(TEMP_COVERAGE_DATA_ROOT_MODULE)/integration_handler.cov
UNIT_APPLICATION_COV_FILE := $(TEMP_COVERAGE_DATA_ROOT_MODULE)/unit_application.cov
INTEGRATION_APPLICATION_COV_FILE := $(TEMP_COVERAGE_DATA_ROOT_MODULE)/integration_application.cov
# The final merged coverage profile file
MERGED_MODULE_COV_FILE := $(TEMP_COVERAGE_DATA_ROOT_MODULE)/combined_leaderboard_module.cov
HTML_REPORT_OUTPUT_DIR := ./reports # Directory for HTML reports

# Target to run leaderboard handler unit tests and get coverage
# Outputs coverage profile to a file using -coverprofile
coverage-unit-leaderboard-handlers:
	@echo "Running leaderboard handler unit tests (outputting coverage profile to $(UNIT_HANDLER_COV_FILE))..."
	-mkdir -p $(TEMP_COVERAGE_DATA_ROOT_MODULE)
	go test \
    $(LEADERBOARD_HANDLER_UNIT_TEST_PKG) \
    -v \
    -cover \
    -covermode=atomic \
    -coverpkg=$(LEADERBOARD_HANDLER_COVERAGE_PKGS) \
    -coverprofile=$(UNIT_HANDLER_COV_FILE) # Output coverage profile to file

# Target to run leaderboard handler integration tests and get coverage
# Outputs coverage profile to a file using -coverprofile
coverage-integration-leaderboard-handlers:
	@echo "Running leaderboard handler integration tests (outputting coverage profile to $(INTEGRATION_HANDLER_COV_FILE))..."
	-mkdir -p $(TEMP_COVERAGE_DATA_ROOT_MODULE)
	go test \
    $(LEADERBOARD_HANDLER_INTEGRATION_TEST_PKG) \
    -v \
    -cover \
    -covermode=atomic \
    -coverpkg=$(LEADERBOARD_MODULE_COVERAGE_PKGS) \
    -coverprofile=$(INTEGRATION_HANDLER_COV_FILE) # Output coverage profile to file

# Target to run leaderboard application unit tests and get coverage
# Outputs coverage profile to a file using -coverprofile
coverage-unit-leaderboard-application:
	@echo "Running leaderboard application unit tests (outputting coverage profile to $(UNIT_APPLICATION_COV_FILE))..."
	-mkdir -p $(TEMP_COVERAGE_DATA_ROOT_MODULE)
	go test \
    $(LEADERBOARD_APPLICATION_UNIT_TEST_PKG) \
    -v \
    -cover \
    -covermode=atomic \
    -coverpkg=$(LEADERBOARD_APPLICATION_COVERAGE_PKGS) \
    -coverprofile=$(UNIT_APPLICATION_COV_FILE) # Output coverage profile to file

# Target to run leaderboard application integration tests and get coverage
# Outputs coverage profile to a file using -coverprofile
coverage-integration-leaderboard-application:
	@echo "Running leaderboard application integration tests (outputting coverage profile to $(INTEGRATION_APPLICATION_COV_FILE))..."
	-mkdir -p $(TEMP_COVERAGE_DATA_ROOT_MODULE)
	go test \
    $(LEADERBOARD_APPLICATION_INTEGRATION_TEST_PKG) \
    -v \
    -cover \
    -covermode=atomic \
    -coverpkg=$(LEADERBOARD_MODULE_COVERAGE_PKGS) \
    -coverprofile=$(INTEGRATION_APPLICATION_COV_FILE) # Output coverage profile to file

# Target to run all leaderboard module tests (unit/integration for handlers/application)
# and combine coverage using text profiles, then print the total percentage.
coverage-leaderboard-module-combined: \
  coverage-unit-leaderboard-handlers \
  coverage-integration-leaderboard-handlers \
  coverage-unit-leaderboard-application \
  coverage-integration-leaderboard-application
	@echo "--- Starting Combined Coverage for Leaderboard Module ---"
	@echo "Cleaning up old combined coverage profile..."
	-rm -f $(MERGED_MODULE_COV_FILE)

	@echo "Merging text coverage profiles..."
  # Create the combined output file with the mode: set header
  echo "mode: set" > $(MERGED_MODULE_COV_FILE)
  # Append the content of each coverage file, excluding its header
  grep -v "^mode:" $(UNIT_HANDLER_COV_FILE) >> $(MERGED_MODULE_COV_FILE)
  grep -v "^mode:" $(INTEGRATION_HANDLER_COV_FILE) >> $(MERGED_MODULE_COV_FILE)
  grep -v "^mode:" $(UNIT_APPLICATION_COV_FILE) >> $(MERGED_MODULE_COV_FILE)
  grep -v "^mode:" $(INTEGRATION_APPLICATION_COV_FILE) >> $(MERGED_MODULE_COV_FILE)

	@echo "Printing overall combined coverage percentage for the leaderboard module..."
  # Use go tool cover -func on the merged profile to show function coverage and total percentage
	go tool cover -func $(MERGED_MODULE_COV_FILE)

	@echo "--- Combined Coverage for Leaderboard Module Finished ---"
	@echo "Combined coverage profile created at $(MERGED_MODULE_COV_FILE)"
	@echo "Run 'make coverage-leaderboard-module-html' to generate an HTML report."

# Target to generate combined HTML coverage report from the merged profile
coverage-leaderboard-module-html: coverage-leaderboard-module-combined
	@echo "Generating combined HTML coverage report..."
	-mkdir -p $(HTML_REPORT_OUTPUT_DIR)
	go tool cover -html $(MERGED_MODULE_COV_FILE) -o $(HTML_REPORT_OUTPUT_DIR)/combined-leaderboard-module.html
	@echo "Combined HTML report generated at $(HTML_REPORT_OUTPUT_DIR)/combined-leaderboard-module.html"


# --- Coverage Targets for Round Module (NEW) ---

# Define package paths for clarity
ROUND_APPLICATION_UNIT_TEST_PKG := ./app/modules/round/application
ROUND_APPLICATION_INTEGRATION_TEST_PKG := ./integration_tests/modules/round/application_tests
ROUND_HANDLER_UNIT_TEST_PKG := ./app/modules/round/infrastructure/handlers
ROUND_HANDLER_INTEGRATION_TEST_PKG := ./integration_tests/modules/round/handler_tests

# Define packages to measure coverage for
# Unit tests should measure coverage of the package being tested.
ROUND_APPLICATION_COVERAGE_PKGS := ./app/modules/round/application
ROUND_HANDLER_COVERAGE_PKGS := ./app/modules/round/infrastructure/handlers
# Integration tests and combined report should measure coverage for relevant packages in the module.
# Including utils and time_utils as they are part of the round module's internal logic.
ROUND_MODULE_COVERAGE_PKGS := ./app/modules/round/application,./app/modules/round/infrastructure/handlers,./app/modules/round/infrastructure/repositories,./app/modules/round/utils,./app/modules/round/time_utils

# Define directories and filenames for coverage data using text profile format
TEMP_COVERAGE_DATA_ROOT_MODULE_ROUND := .coverage_temp/round_module
UNIT_ROUND_APPLICATION_COV_FILE := $(TEMP_COVERAGE_DATA_ROOT_MODULE_ROUND)/unit_application.cov
INTEGRATION_ROUND_APPLICATION_COV_FILE := $(TEMP_COVERAGE_DATA_ROOT_MODULE_ROUND)/integration_application.cov
UNIT_ROUND_HANDLER_COV_FILE := $(TEMP_COVERAGE_DATA_ROOT_MODULE_ROUND)/unit_handler.cov
INTEGRATION_ROUND_HANDLER_COV_FILE := $(TEMP_COVERAGE_DATA_ROOT_MODULE_ROUND)/integration_handler.cov
# The final merged coverage profile file
MERGED_ROUND_MODULE_COV_FILE := $(TEMP_COVERAGE_DATA_ROOT_MODULE_ROUND)/combined_round_module.cov

# Target to run round application unit tests and get coverage
# Outputs coverage profile to a file using -coverprofile
coverage-unit-round-application:
	@echo "Running round application unit tests (outputting coverage profile to $(UNIT_ROUND_APPLICATION_COV_FILE))..."
	-mkdir -p $(TEMP_COVERAGE_DATA_ROOT_MODULE_ROUND)
	go test \
    $(ROUND_APPLICATION_UNIT_TEST_PKG) \
    -v \
    -cover \
    -covermode=atomic \
    -coverpkg=$(ROUND_APPLICATION_COVERAGE_PKGS) \
    -coverprofile=$(UNIT_ROUND_APPLICATION_COV_FILE) # Output coverage profile to file

# Target to run round application integration tests and get coverage
# Outputs coverage profile to a file using -coverprofile
coverage-integration-round-application:
	@echo "Running round application integration tests (outputting coverage profile to $(INTEGRATION_ROUND_APPLICATION_COV_FILE))..."
	-mkdir -p $(TEMP_COVERAGE_DATA_ROOT_MODULE_ROUND)
	go test \
    $(ROUND_APPLICATION_INTEGRATION_TEST_PKG) \
    -v \
    -cover \
    -covermode=atomic \
    -coverpkg=$(ROUND_MODULE_COVERAGE_PKGS) \
    -coverprofile=$(INTEGRATION_ROUND_APPLICATION_COV_FILE) # Output coverage profile to file

# Target to run round handler unit tests and get coverage
# Outputs coverage profile to a file using -coverprofile
coverage-unit-round-handlers:
	@echo "Running round handler unit tests (outputting coverage profile to $(UNIT_ROUND_HANDLER_COV_FILE))..."
	-mkdir -p $(TEMP_COVERAGE_DATA_ROOT_MODULE_ROUND)
	go test \
    $(ROUND_HANDLER_UNIT_TEST_PKG) \
    -v \
    -cover \
    -covermode=atomic \
    -coverpkg=$(ROUND_HANDLER_COVERAGE_PKGS) \
    -coverprofile=$(UNIT_ROUND_HANDLER_COV_FILE) # Output coverage profile to file

# Target to run round handler integration tests and get coverage
# Outputs coverage profile to a file using -coverprofile
coverage-integration-round-handlers:
	@echo "Running round handler integration tests (outputting coverage profile to $(INTEGRATION_ROUND_HANDLER_COV_FILE))..."
	-mkdir -p $(TEMP_COVERAGE_DATA_ROOT_MODULE_ROUND)
	go test \
    $(ROUND_HANDLER_INTEGRATION_TEST_PKG) \
    -v \
    -cover \
    -covermode=atomic \
    -coverpkg=$(ROUND_MODULE_COVERAGE_PKGS) \
    -coverprofile=$(INTEGRATION_ROUND_HANDLER_COV_FILE) # Output coverage profile to file

# Target to run all round module tests (unit/integration for handlers/application)
# and combine coverage using text profiles, then print the total percentage.
coverage-round-module-combined: \
  coverage-unit-round-handlers \
  coverage-integration-round-handlers \
  coverage-unit-round-application \
  coverage-integration-round-application
	@echo "--- Starting Combined Coverage for Round Module ---"
	@echo "Cleaning up old combined coverage profile..."
	-rm -f $(MERGED_ROUND_MODULE_COV_FILE)

	@echo "Merging text coverage profiles..."
  # Create the combined output file with the mode: set header
  echo "mode: set" > $(MERGED_ROUND_MODULE_COV_FILE)
  # Append the content of each coverage file, excluding its header
  grep -v "^mode:" $(UNIT_ROUND_HANDLER_COV_FILE) >> $(MERGED_ROUND_MODULE_COV_FILE)
  grep -v "^mode:" $(INTEGRATION_ROUND_HANDLER_COV_FILE) >> $(MERGED_ROUND_MODULE_COV_FILE)
  grep -v "^mode:" $(UNIT_ROUND_APPLICATION_COV_FILE) >> $(MERGED_ROUND_MODULE_COV_FILE)
  grep -v "^mode:" $(INTEGRATION_ROUND_APPLICATION_COV_FILE) >> $(MERGED_ROUND_MODULE_COV_FILE)

	@echo "Printing overall combined coverage percentage for the round module..."
  # Use go tool cover -func on the merged profile to show function coverage and total percentage
	go tool cover -func $(MERGED_ROUND_MODULE_COV_FILE)

	@echo "--- Combined Coverage for Round Module Finished ---"
	@echo "Combined coverage profile created at $(MERGED_ROUND_MODULE_COV_FILE)"
	@echo "Run 'make coverage-round-module-html' to generate an HTML report."

# Target to generate combined HTML coverage report from the merged profile
coverage-round-module-html: coverage-round-module-combined
	@echo "Generating combined HTML coverage report..."
	-mkdir -p $(HTML_REPORT_OUTPUT_DIR)
	go tool cover -html $(MERGED_ROUND_MODULE_COV_FILE) -o $(HTML_REPORT_OUTPUT_DIR)/combined-round-module.html
	@echo "Combined HTML report generated at $(HTML_REPORT_OUTPUT_DIR)/combined-round-module.html"


# Clean up temporary coverage data files and directories
clean-coverage:
	@echo "Cleaning up temporary coverage data..."
	-rm -rf $(TEMP_COVERAGE_DATA_ROOT_MODULE) $(TEMP_COVERAGE_DATA_ROOT_MODULE_ROUND)
	-rm -f combined-leaderboard-handlers.out # Clean up old text file if it exists
	-rm -f combined-leaderboard-module.out # Clean up old text file if it exists
	-rm -f combined-round-module.out # Clean up old text file if it exists

# --- Mock Generation Targets ---
MOCKGEN := mockgen
USER_DIR := ./app/modules/user
LB_DIR := ./app/modules/leaderboard
ROUND_DIR := ./app/modules/round
EVENTBUS_DIR := ./app/eventbus
SCORE_DIR := ./app/modules/score
SHARED_MODULE := github.com/Black-And-White-Club/frolf-bot-shared

# User module mock generation
mocks-user:
  $(MOCKGEN) -source=$(USER_DIR)/application/interface.go -destination=$(USER_DIR)/application/mocks/mock_service.go -package=mocks
  $(MOCKGEN) -source=$(USER_DIR)/infrastructure/handlers/interface.go -destination=$(USER_DIR)/infrastructure/handlers/mocks/mock_handlers.go -package=mocks
  $(MOCKGEN) -source=$(USER_DIR)/infrastructure/router/interface.go -destination=$(USER_DIR)/infrastructure/router/mocks/mock_router.go -package=mocks
  $(MOCKGEN) -source=$(USER_DIR)/infrastructure/repositories/interface.go -destination=$(USER_DIR)/infrastructure/repositories/mocks/mock_db.go -package=mocks

# Leaderboard module mock generation
mocks-leaderboard:
  $(MOCKGEN) -source=$(LB_DIR)/application/interface.go -destination=$(LB_DIR)/application/mocks/mock_service.go -package=mocks
  $(MOCKGEN) -source=$(LB_DIR)/infrastructure/handlers/interface.go -destination=$(LB_DIR)/infrastructure/handlers/mocks/mock_handlers.go -package=mocks
  $(MOCKGEN) -source=$(LB_DIR)/infrastructure/router/interface.go -destination=$(LB_DIR)/infrastructure/router/mocks/mock_router.go -package=mocks
  $(MOCKGEN) -source=$(LB_DIR)/infrastructure/repositories/interface.go -destination=$(LB_DIR)/infrastructure/repositories/mocks/mock_db.go -package=mocks

# Round module mock generation
mocks-round:
  $(MOCKGEN) -source=$(ROUND_DIR)/application/interface.go -destination=$(ROUND_DIR)/application/mocks/mock_service.go -package=mocks
  $(MOCKGEN) -source=$(ROUND_DIR)/infrastructure/handlers/interface.go -destination=$(ROUND_DIR)/infrastructure/handlers/mocks/mock_handlers.go -package=mocks
  $(MOCKGEN) -source=$(ROUND_DIR)/infrastructure/router/interface.go -destination=$(ROUND_DIR)/infrastructure/router/mocks/mock_router.go -package=mocks
  $(MOCKGEN) -source=$(ROUND_DIR)/infrastructure/repositories/interface.go -destination=$(ROUND_DIR)/infrastructure/repositories/mocks/mock_db.go -package=mocks
# Round Utils Mock generation
  $(MOCKGEN) -source=$(ROUND_DIR)/utils/clock.go -destination=$(ROUND_DIR)/mocks/mock_clock.go -package=mocks
  $(MOCKGEN) -source=$(ROUND_DIR)/time_utils/time_conversion.go -destination=$(ROUND_DIR)/mocks/mock_conversion.go -package=mocks
  $(MOCKGEN) -source=$(ROUND_DIR)/utils/validator.go -destination=$(ROUND_DIR)/mocks/mock_validator.go -package=mocks

# Score module mock generation
mocks-score:
  $(MOCKGEN) -source=$(SCORE_DIR)/application/interface.go -destination=$(SCORE_DIR)/application/mocks/mock_service.go -package=mocks
  $(MOCKGEN) -source=$(SCORE_DIR)/infrastructure/handlers/interface.go -destination=$(SCORE_DIR)/infrastructure/handlers/mocks/mock_handlers.go -package=mocks
  $(MOCKGEN) -source=$(SCORE_DIR)/infrastructure/router/interface.go -destination=$(SCORE_DIR)/infrastructure/router/mocks/mock_router.go -package=mocks
  $(MOCKGEN) -source=$(SCORE_DIR)/infrastructure/repositories/interface.go -destination=$(SCORE_DIR)/infrastructure/repositories/mocks/mock_db.go -package=mocks

# EventBus mock generation
mocks-eventbus:
  $(MOCKGEN) -source=../frolf-bot-shared/eventbus/eventbus.go -destination=$(EVENTBUS_DIR)/mocks/mock_eventbus.go -package=mocks

# Generate all mocks for user and eventbus
mocks-all: mocks-user mocks-eventbus mocks-leaderboard mocks-round mocks-score

# Define LDFLAGS for the build-version target using a target-specific variable
# This variable is evaluated when the Makefile is read.
build_version_ldflags := -X 'main.Version=$(shell git describe --tags --always)'

build-version:
	@echo "Building with version information..."
  # The 'go build' command will use the 'build_version_ldflags' variable
  go build -ldflags="$(build_version_ldflags)" ./...
