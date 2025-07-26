.PHONY: help build test test-integration benchmark clean docker-build docker-run docker-stop lint fmt proto

# Default target
help:
	@echo "Available targets:"
	@echo "  build          - Build the scheduler binary"
	@echo "  test           - Run unit tests"
	@echo "  test-integration - Run integration tests"
	@echo "  benchmark      - Run benchmarks"
	@echo "  clean          - Clean build artifacts"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run with Docker Compose"
	@echo "  docker-stop    - Stop Docker Compose"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  proto          - Generate protobuf code"
	@echo "  cli            - Build CLI tool"

# Build the scheduler binary
build:
	@echo "Building scheduler..."
	go build -o bin/scheduler cmd/scheduler/main.go
	@echo "Building CLI tool..."
	go build -o bin/cli cmd/cli/main.go
	@echo "Build complete!"

# Run unit tests
test:
	@echo "Running unit tests..."
	go test -v ./internal/...
	@echo "Unit tests complete!"

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./internal/scheduler/
	@echo "Integration tests complete!"

# Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./internal/queue/
	@echo "Benchmarks complete!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf data/
	@echo "Clean complete!"

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t task-scheduler .
	@echo "Docker build complete!"

# Run with Docker Compose
docker-run:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d
	@echo "Services started! Access at:"
	@echo "  Node 1 HTTP API: http://localhost:8080"
	@echo "  Node 2 HTTP API: http://localhost:8082"
	@echo "  Node 3 HTTP API: http://localhost:8084"
	@echo "  Redis: localhost:6379"

# Stop Docker Compose
docker-stop:
	@echo "Stopping services..."
	docker-compose down
	@echo "Services stopped!"

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...
	@echo "Linting complete!"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "Formatting complete!"

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/scheduler.proto
	@echo "Protobuf generation complete!"

# Build CLI tool
cli:
	@echo "Building CLI tool..."
	go build -o bin/cli cmd/cli/main.go
	@echo "CLI build complete!"

# Run local development
dev:
	@echo "Starting local development..."
	@echo "Make sure Redis is running on localhost:6379"
	go run cmd/scheduler/main.go -node-id=dev-node-1 -http-addr=localhost:8080 -grpc-addr=localhost:9090 -redis-addr=localhost:6379

# Run multiple nodes locally
dev-cluster:
	@echo "Starting local cluster..."
	@echo "Starting node 1..."
	go run cmd/scheduler/main.go -node-id=node-1 -http-addr=localhost:8080 -grpc-addr=localhost:9090 -redis-addr=localhost:6379 -postgres-dsn=postgres://scheduler:scheduler_pass@localhost:5432/task_scheduler?sslmode=disable &
	@echo "Starting node 2..."
	go run cmd/scheduler/main.go -node-id=node-2 -http-addr=localhost:8082 -grpc-addr=localhost:9091 -redis-addr=localhost:6379 -postgres-dsn=postgres://scheduler:scheduler_pass@localhost:5432/task_scheduler?sslmode=disable -peers=localhost:8081 &
	@echo "Starting node 3..."
	go run cmd/scheduler/main.go -node-id=node-3 -http-addr=localhost:8084 -grpc-addr=localhost:9092 -redis-addr=localhost:6379 -postgres-dsn=postgres://scheduler:scheduler_pass@localhost:5432/task_scheduler?sslmode=disable -peers=localhost:8081,localhost:8083 &
	@echo "Starting node 4..."
	go run cmd/scheduler/main.go -node-id=node-4 -http-addr=localhost:8086 -grpc-addr=localhost:9093 -redis-addr=localhost:6379 -postgres-dsn=postgres://scheduler:scheduler_pass@localhost:5432/task_scheduler?sslmode=disable -peers=localhost:8081,localhost:8083,localhost:8085 &
	@echo "Local cluster started!"
	@echo "Access nodes at:"
	@echo "  Node 1: http://localhost:8080"
	@echo "  Node 2: http://localhost:8082"
	@echo "  Node 3: http://localhost:8084"
	@echo "  Node 4: http://localhost:8086"

# Stop local cluster
dev-stop:
	@echo "Stopping local cluster..."
	pkill -f "go run cmd/scheduler/main.go"
	@echo "Local cluster stopped!"

# Test CLI
test-cli:
	@echo "Testing CLI..."
	@echo "Health check:"
	./bin/cli -command=health
	@echo ""
	@echo "Cluster info:"
	./bin/cli -command=cluster
	@echo ""
	@echo "Stats:"
	./bin/cli -command=stats
	@echo ""
	@echo "Submit task:"
	./bin/cli -command=submit -priority=high -payload='{"message": "test task"}'
	@echo ""
	@echo "List tasks:"
	./bin/cli -command=tasks

# Performance test
perf-test:
	@echo "Running performance test..."
	@echo "Submitting 1000 tasks..."
	for i in {1..1000}; do \
		curl -s -X POST http://localhost:8080/api/v1/tasks \
			-H "Content-Type: application/json" \
			-d '{"priority": "medium", "payload": {"id": "'$$i'", "message": "perf test"}}' > /dev/null; \
	done
	@echo "Performance test complete!"

# Load test
load-test:
	@echo "Running load test..."
	@echo "This will submit tasks continuously for 60 seconds..."
	timeout 60s bash -c 'while true; do \
		curl -s -X POST http://localhost:8080/api/v1/tasks \
			-H "Content-Type: application/json" \
			-d "{\"priority\": \"low\", \"payload\": {\"timestamp\": \"$$(date +%s)\", \"message\": \"load test\"}}" > /dev/null; \
		sleep 0.1; \
	done'
	@echo "Load test complete!"

# Show logs
logs:
	@echo "Showing Docker Compose logs..."
	docker-compose logs -f

# Show specific service logs
logs-node1:
	docker-compose logs -f scheduler-node-1

logs-node2:
	docker-compose logs -f scheduler-node-2

logs-node3:
	docker-compose logs -f scheduler-node-3

logs-node4:
	docker-compose logs -f scheduler-node-4

logs-redis:
	docker-compose logs -f redis

logs-postgres:
	docker-compose logs -f postgres

# Reset everything
reset:
	@echo "Resetting everything..."
	docker-compose down -v
	rm -rf data/
	rm -rf bin/
	@echo "Reset complete!"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download
	@echo "Dependencies installed!"

# Check if Redis is running
check-redis:
	@echo "Checking Redis connection..."
	@if redis-cli ping > /dev/null 2>&1; then \
		echo "Redis is running"; \
	else \
		echo "Redis is not running. Please start Redis first."; \
		exit 1; \
	fi

# Full setup
setup: deps proto build
	@echo "Setup complete!"

# Full test suite
test-all: test test-integration benchmark
	@echo "All tests complete!"

# Development workflow
dev-setup: check-redis build
	@echo "Development setup complete!" 