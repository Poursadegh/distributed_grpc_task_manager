.PHONY: build test clean run docker-build docker-run help

# Default target
help:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  test         - Run tests"
	@echo "  test-bench   - Run benchmarks"
	@echo "  clean        - Clean build artifacts"
	@echo "  run          - Run the application"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run with Docker Compose"
	@echo "  lint         - Run linter"
	@echo "  fmt          - Format code"

# Build the application
build:
	go build -o bin/scheduler ./cmd/scheduler

# Run tests
test:
	go test -v ./...

# Run benchmarks
test-bench:
	go test -bench=. ./internal/queue/

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Run the application
run: build
	./bin/scheduler

# Build Docker image
docker-build:
	docker build -t task-scheduler .

# Run with Docker Compose
docker-run:
	docker-compose up -d

# Stop Docker Compose
docker-stop:
	docker-compose down

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...
	go vet ./...

# Install dependencies
deps:
	go mod tidy
	go mod download

# Generate mocks (if using mockgen)
mocks:
	mockgen -source=internal/storage/storage.go -destination=internal/storage/mock_storage.go

# Run integration tests
test-integration:
	go test -tags=integration ./...

# Show coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Development setup
dev-setup: deps fmt lint test

# Production build
build-prod:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/scheduler ./cmd/scheduler 