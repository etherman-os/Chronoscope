.PHONY: up down proto test lint

# Docker Compose
up:
	docker compose -f docker/docker-compose.yml up -d

down:
	docker compose -f docker/docker-compose.yml down

# Protocol Buffers
default_proto_dir := protocols/capture-schema
swift_out := packages/sdk-macos/Sources/Chronoscope/Core

default_proto := $(default_proto_dir)/session.proto

proto:
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
	@echo "Running ingestion service tests..."
	cd services/ingestion && go test ./...
	@echo "Running analytics service tests..."
	cd services/analytics && go test ./...
	@echo "Running SDK macOS tests..."
	cd packages/sdk-macos && swift test

# Linting
lint:
	@echo "Running golangci-lint on ingestion service..."
	cd services/ingestion && golangci-lint run ./...
	@echo "Running golangci-lint on analytics service..."
	cd services/analytics && golangci-lint run ./...
