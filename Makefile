.PHONY: all build clean proto server client webserver help

# Variables
BINARY_DIR=bin
PROTO_DIR=api/proto/v1
PKG_DIR=pkg/auction

all: proto build

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	protoc --proto_path=$(PROTO_DIR) \
		--go_out=$(PKG_DIR) \
		--go-grpc_out=$(PKG_DIR) \
		$(PROTO_DIR)/auction.proto

# Build all binaries
build: server client webserver

# Build server
server:
	@echo "Building server..."
	go build -o $(BINARY_DIR)/auction-server.exe cmd/server/main.go

# Build CLI client
client:
	@echo "Building CLI client..."
	go build -o $(BINARY_DIR)/auction-client.exe cmd/client/main.go

# Build web server
webserver:
	@echo "Building web server..."
	go build -o $(BINARY_DIR)/webserver.exe cmd/webserver/main.go

# Run server
run-server:
	go run cmd/server/main.go

# Run web server
run-web:
	go run cmd/webserver/main.go

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BINARY_DIR)/*
	go clean

# Run tests
test:
	go test ./...

# Show help
help:
	@echo "Available targets:"
	@echo "  make proto       - Generate protobuf code"
	@echo "  make build       - Build all binaries"
	@echo "  make server      - Build gRPC server"
	@echo "  make client      - Build CLI client"
	@echo "  make webserver   - Build web server"
	@echo "  make run-server  - Run gRPC server"
	@echo "  make run-web     - Run web server"
	@echo "  make clean       - Remove build artifacts"
	@echo "  make test        - Run tests"