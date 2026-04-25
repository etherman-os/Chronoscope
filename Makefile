.PHONY: up down proto test lint

# Docker Compose
up:
	docker compose -f docker/docker-compose.yml up -d --build

down:
	docker compose -f docker/docker-compose.yml down

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
	cd services/ingestion && go test ./... || failed=1; \
	echo "Running analytics service tests..."; \
	cd services/analytics && go test ./... || failed=1; \
	echo "Running SDK macOS tests..."; \
	cd packages/sdk-macos && swift test || failed=1; \
	exit $$failed

# Linting
lint:
	@echo "Running golangci-lint on ingestion service..."
	cd services/ingestion && golangci-lint run ./...
	@echo "Running golangci-lint on analytics service..."
	cd services/analytics && golangci-lint run ./...
