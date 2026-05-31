DATABASE_URL ?= postgres://pav:pav@localhost:5432/pav?sslmode=disable
MIGRATE ?= migrate
CLAIM_ID ?= 00000000-0000-4000-8000-000000000001

.PHONY: db-up db-down migrate-up migrate-down seed run-rules run-template compare test build fmt vet

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

migrate-up:
	@if command -v migrate >/dev/null 2>&1; then \
		$(MIGRATE) -path migrations -database "$(DATABASE_URL)" up; \
	else \
		docker run --rm --network pav_default -v "$(CURDIR)/migrations:/migrations" migrate/migrate \
			-path=/migrations -database "postgres://pav:pav@postgres:5432/pav?sslmode=disable" up; \
	fi

migrate-down:
	@if command -v migrate >/dev/null 2>&1; then \
		$(MIGRATE) -path migrations -database "$(DATABASE_URL)" down -all; \
	else \
		docker run --rm --network pav_default -v "$(CURDIR)/migrations:/migrations" migrate/migrate \
			-path=/migrations -database "postgres://pav:pav@postgres:5432/pav?sslmode=disable" down -all; \
	fi

seed:
	docker exec -i pav-postgres-1 psql -U pav -d pav < seeds/dev/seed.sql

run-rules:
	go run ./cmd/rules-engine

run-template:
	go run ./cmd/template-engine

compare:
	@./scripts/compare.sh $(CLAIM_ID)

build:
	go build ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./... -count=1
