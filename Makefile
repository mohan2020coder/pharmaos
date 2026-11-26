.PHONY: help test migrate seed lint fmt dev up down test-db test-all mobile-typecheck admin-web-typecheck admin-web-build admin-web-audit deps-audit scan-secrets load-smoke backup-drill verify-release

help:
	@echo "Targets: up, down, dev, migrate, seed, test, test-db, test-all, fmt, lint, mobile-typecheck, admin-web-typecheck, admin-web-build, admin-web-audit, deps-audit, scan-secrets, load-smoke, backup-drill, verify-release"

up:
	docker compose -f infrastructure/docker-compose.yml up -d postgres redis

down:
	docker compose -f infrastructure/docker-compose.yml down

dev: up
	cd backend && go run ./cmd/api

migrate:
	cd backend && go run ./cmd/api -migrate-down 0 || true

seed:
	cd backend && go run ./cmd/seed

test:
	cd backend && go test ./...

test-db:
	docker compose -f infrastructure/docker-compose.yml --profile test up -d postgres_test
	cd backend && TEST_DATABASE_URL=postgres://pharmaos:pharmaos@localhost:55433/pharmaos_test?sslmode=disable go test ./...

test-all: fmt lint test

fmt:
	cd backend && gofmt -w ./cmd ./internal ./tests

lint:
	cd backend && go vet ./...

mobile-typecheck:
	cd mobile && npm run typecheck

admin-web-typecheck:
	cd admin-web && npm run typecheck

admin-web-build:
	cd admin-web && npm run build

admin-web-audit:
	cd admin-web && npm run audit

deps-audit:
	cd backend && go mod verify && go list -m -json all

scan-secrets:
	bash scripts/scan-secrets.sh .

load-smoke:
	bash scripts/load-smoke.sh

backup-drill:
	bash scripts/backup-drill.sh

verify-release: fmt lint test mobile-typecheck admin-web-typecheck admin-web-audit deps-audit scan-secrets
