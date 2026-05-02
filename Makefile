.PHONY: up down proto test lint seed-local run-ingestion run-analytics run-processor run-web record-linux demo-session

COMPOSE := $(shell if docker compose version >/dev/null 2>&1; then echo "docker compose"; elif command -v docker-compose >/dev/null 2>&1; then echo "docker-compose"; else echo "docker compose"; fi)

# Docker Compose
up:
	$(COMPOSE) -f docker/docker-compose.yml up -d --build

down:
	$(COMPOSE) -f docker/docker-compose.yml down

seed-local:
	docker exec chronoscope-postgres psql -U chronoscope -d chronoscope -v ON_ERROR_STOP=1 -c "INSERT INTO organizations (id, name, plan) VALUES ('11111111-1111-1111-1111-111111111111', 'Local Chronoscope', 'free') ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name; INSERT INTO projects (id, org_id, name, api_key_hash, retention_days) VALUES ('22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111', 'Local Desktop App', 'ed5a18fb8f807f996d649e379d3f35f39c543a91bdbf88c492f2ebd10d4df86c', 30) ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, api_key_hash = EXCLUDED.api_key_hash;"

run-ingestion:
	cd services/ingestion && set -a && . ./.env.example && set +a && go run cmd/server/main.go

run-analytics:
	cd services/analytics && set -a && . ./.env.example && set +a && go run cmd/server/main.go

run-processor:
	cd services/processor && set -a && . ./.env.example && set +a && cargo run

run-web:
	cd services/web && npm install && npm run dev

API_KEY ?= local-dev-key
ENDPOINT ?= http://localhost:8080
USER_ID ?= local-user
FPS ?= 5
DURATION ?= 30

record-linux:
	cd packages/sdk-linux && cargo run --bin chronoscope-linux-record -- --endpoint $(ENDPOINT) --api-key $(API_KEY) --user-id $(USER_ID) --fps $(FPS) --duration $(DURATION)

DEMO_FRAMES ?= 45

demo-session:
	cd packages/sdk-linux && cargo run --bin chronoscope-demo-session -- --endpoint $(ENDPOINT) --api-key $(API_KEY) --user-id demo-user --fps $(FPS) --frames $(DEMO_FRAMES)

# Protocol Buffers
default_proto_dir := protocols/capture-schema
swift_out := packages/sdk-macos/Sources/Chronoscope/Core

default_proto := $(default_proto_dir)/session.proto

proto:
	@command -v protoc >/dev/null 2>&1 || { echo "protoc is required but not installed"; exit 1; }
	@command -v protoc-gen-go >/dev/null 2>&1 || { echo "protoc-gen-go is required but not installed"; exit 1; }
	@command -v protoc-gen-go-grpc >/dev/null 2>&1 || { echo "protoc-gen-go-grpc is required but not installed"; exit 1; }
	@command -v protoc-gen-swift >/dev/null 2>&1 || { echo "protoc-gen-swift is required but not installed"; exit 1; }
	@echo "Generating Go code from protobuf..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		-I $(default_proto_dir) \
		$(default_proto)
	@echo "Generating Swift code from protobuf..."
	protoc --swift_out=$(swift_out) \
		-I $(default_proto_dir) \
		$(default_proto)

# Testing
test:
	@failed=0; \
	echo "Running ingestion service tests..."; \
	(cd services/ingestion && go test ./...) || failed=1; \
	echo "Running analytics service tests..."; \
	(cd services/analytics && go test ./...) || failed=1; \
	echo "Running shared middleware tests..."; \
	(cd pkg/middleware && go test ./...) || failed=1; \
	echo "Running web dashboard tests..."; \
	(cd services/web && npm test) || failed=1; \
	echo "Running processor tests..."; \
	(cd services/processor && cargo test --locked) || failed=1; \
	echo "Running privacy engine tests..."; \
	(cd services/privacy-engine && cargo test --locked) || failed=1; \
	echo "Running Linux SDK tests..."; \
	(cd packages/sdk-linux && cargo test --locked) || failed=1; \
	if [ "$$(uname)" = "Darwin" ]; then \
		echo "Running SDK macOS tests..."; \
		(cd packages/sdk-macos && swift test) || failed=1; \
	else \
		echo "Skipping SDK macOS tests (macOS only)..."; \
	fi; \
	exit $$failed

# Linting
lint:
	@echo "Running golangci-lint on ingestion service..."
	cd services/ingestion && golangci-lint run ./...
	@echo "Running golangci-lint on analytics service..."
	cd services/analytics && golangci-lint run ./...
