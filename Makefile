.PHONY: migrate-init migrate migrate-all rollback-all

migrate-init:
	go run cmd/bun/main.go migrate init

migrate:
	go run cmd/bun/main.go migrate migrate

migrate-all: migrate-init migrate

rollback-all:
	go run cmd/bun/main.go migrate rollback

run:
	go run cmd/app/main.go

# Directories
MOCKGEN := mockgen
USER_DIR := ./app/modules/user
LB_DIR := ./app/modules/leaderboard
EVENTBUS_DIR := ./app/eventbus

# User module mock generation
mocks-user:
	$(MOCKGEN) -source=$(USER_DIR)/application/interface.go -destination=$(USER_DIR)/application/mocks/mock_service.go -package=mocks
	$(MOCKGEN) -source=$(USER_DIR)/domain/types/types.go -destination=$(USER_DIR)/domain/types/mocks/mock_types.go -package=mocks
	$(MOCKGEN) -source=$(USER_DIR)/infrastructure/handlers/interface.go -destination=$(USER_DIR)/infrastructure/handlers/mocks/mock_handlers.go -package=mocks
	$(MOCKGEN) -source=$(USER_DIR)/infrastructure/router/interface.go -destination=$(USER_DIR)/infrastructure/router/mocks/mock_router.go -package=mocks
	$(MOCKGEN) -source=$(USER_DIR)/infrastructure/repositories/interface.go -destination=$(USER_DIR)/infrastructure/repositories/mocks/mock_db.go -package=mocks

# User module mock generation
mocks-leaderboard:
	$(MOCKGEN) -source=$(LB_DIR)/application/interface.go -destination=$(LB_DIR)/application/mocks/mock_service.go -package=mocks
	$(MOCKGEN) -source=$(LB_DIR)/domain/types/types.go -destination=$(LB_DIR)/domain/types/mocks/mock_types.go -package=mocks
	$(MOCKGEN) -source=$(LB_DIR)/infrastructure/handlers/interface.go -destination=$(LB_DIR)/infrastructure/handlers/mocks/mock_handlers.go -package=mocks
	$(MOCKGEN) -source=$(LB_DIR)/infrastructure/router/interface.go -destination=$(LB_DIR)/infrastructure/router/mocks/mock_router.go -package=mocks
	$(MOCKGEN) -source=$(LB_DIR)/infrastructure/repositories/interface.go -destination=$(LB_DIR)/infrastructure/repositories/mocks/mock_db.go -package=mocks

# EventBus mock generation
mocks-eventbus:
	$(MOCKGEN) -source=./app/shared/eventbus.go -destination=$(EVENTBUS_DIR)/mocks/mock_eventbus.go -package=mocks

# Generate all mocks for user and eventbus
mocks-all: mocks-user mocks-eventbus mocks-leaderboard
