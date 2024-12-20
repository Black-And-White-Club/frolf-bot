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
