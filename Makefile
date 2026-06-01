DATABASE_URL ?= postgres://pav:pav@localhost:5432/pav?sslmode=disable
MIGRATE ?= migrate
CLAIM_ID ?= 00000000-0000-4000-8000-000000000001

.PHONY: db-up db-down migrate-up migrate-down seed seed-configs run-rules run-template compare test build fmt vet
.PHONY: localstack-up localstack-down invoke-transformer sam-deploy-localstack run-outbound-workflow start-outbound-sfn enqueue-claim

db-up:
	docker compose up -d postgres

localstack-up:
	chmod +x scripts/localstack-init.sh
	docker compose up -d postgres redis localstack

localstack-down:
	docker compose stop redis localstack

invoke-transformer:
	chmod +x scripts/invoke-transformer-local.sh
	./scripts/invoke-transformer-local.sh

sam-deploy-localstack:
	chmod +x scripts/deploy-localstack.sh
	./scripts/deploy-localstack.sh

run-outbound-workflow:
	chmod +x scripts/run-outbound-workflow-local.sh
	./scripts/run-outbound-workflow-local.sh

start-outbound-sfn:
	chmod +x scripts/start-outbound-sfn-localstack.sh
	./scripts/start-outbound-sfn-localstack.sh

enqueue-claim:
	chmod +x scripts/enqueue-claim-localstack.sh
	./scripts/enqueue-claim-localstack.sh

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

seed: seed-data seed-configs

seed-data:
	docker exec -i pav-postgres-1 psql -U pav -d pav < seeds/dev/seed.sql

seed-configs:
	chmod +x scripts/seed-configs.sh
	./scripts/seed-configs.sh

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
