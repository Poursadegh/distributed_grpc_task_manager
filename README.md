# Distributed Task Scheduler

A production-ready distributed task scheduler implemented in Go that handles prioritized tasks across multiple worker nodes, ensuring scalability, resilience, and observability.

## Overview

This distributed task scheduler provides:
- Priority queue with High/Medium/Low levels
- gRPC API for task submission and monitoring
- Raft-based leader election for distributed coordination
- Redis persistence for task durability
- Worker pool with configurable concurrency
- Prometheus metrics for monitoring

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Node 1        │    │   Node 2        │    │   Node 3        │
│  (Leader)       │    │  (Follower)     │    │  (Follower)     │
│                 │    │                 │    │                 │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │   gRPC API  │ │    │ │   gRPC API  │ │    │ │   gRPC API  │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │Distributed  │ │    │ │Distributed  │ │    │ │Distributed  │ │
│ │Coordinator  │ │    │ │Coordinator  │ │    │ │Coordinator  │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │  Scheduler  │ │    │ │  Scheduler  │ │    │ │  Scheduler  │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │Worker Pool  │ │    │ │Worker Pool  │ │    │ │Worker Pool  │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │Priority Q   │ │    │ │Priority Q   │ │    │ │Priority Q   │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │   Raft      │ │    │ │   Raft      │ │    │ │   Raft      │ │
│ │ Consensus   │ │    │ │ Consensus   │ │    │ │ Consensus   │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │      Redis      │
                    │   (Storage)     │
                    └─────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- Redis (for production)

### Local Development

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd task-scheduler
   ```

2. **Install dependencies**
   ```bash
   go mod tidy
   ```

3. **Generate protobuf code**
   ```bash
   make proto
   ```

4. **Run with Docker Compose**
   ```bash
   docker-compose up -d
   ```

5. **Submit a task**
   ```bash
   curl -X POST http://localhost:9090/api/v1/tasks \
     -H "Content-Type: application/json" \
     -d '{
       "priority": "high",
       "payload": {"message": "Hello, World!"}
     }'
   ```

6. **Check task status**
   ```bash
   curl http://localhost:9090/api/v1/tasks
   ```

### Manual Setup

1. **Start Redis**
   ```bash
   redis-server
   ```

2. **Run the scheduler**
   ```bash
   go run cmd/scheduler/main.go \
     -node-id=node-1 \
     -grpc-addr=localhost:9090 \
     -raft-addr=localhost:8081 \
     -redis-addr=localhost:6379
   ```

## API Reference

### Task Endpoints

#### Submit Task
```http
POST /api/v1/tasks
Content-Type: application/json

{
  "priority": "high|medium|low",
  "payload": {...}
}
```

#### Get All Tasks
```http
GET /api/v1/tasks
```

#### Get Task by ID
```http
GET /api/v1/tasks/{id}
```

#### Get Tasks by Status
```http
GET /api/v1/tasks/status/{status}
```

### Cluster Endpoints

#### Get Cluster Info
```http
GET /api/v1/cluster/info
```

#### Get Statistics
```http
GET /api/v1/cluster/stats
```

### Health Check
```http
GET /health
```

### Metrics
```http
GET /metrics
```

## Configuration

### Command Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-node-id` | `node-1` | Unique node identifier |
| `-grpc-addr` | `localhost:9090` | gRPC server address |
| `-raft-addr` | `localhost:8081` | Raft server address |
| `-redis-addr` | `localhost:6379` | Redis server address |
| `-peers` | `` | Comma-separated list of peer addresses |

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NODE_ID` | `node-1` | Node identifier |
| `GRPC_ADDRESS` | `:9090` | gRPC server address |
| `RAFT_ADDRESS` | `:8081` | Raft server address |
| `REDIS_ADDRESS` | `localhost:6379` | Redis server address |

## Testing

### Run Unit Tests
```bash
go test ./...
```

### Run Benchmarks
```bash
go test -bench=. ./internal/queue/
```

### Run Integration Tests
```bash
go test -tags=integration ./...
```

## Performance

### Expected Performance

- **Task Submission**: ~1000 tasks/second per node
- **Task Processing**: ~500 tasks/second per worker
- **Queue Operations**: O(log n) for add/remove operations
- **Latency**: <10ms for queue operations, <100ms for task submission

### Scalability

- **Horizontal**: Add more nodes to increase throughput
- **Vertical**: Increase worker count per node
- **Storage**: Redis cluster for high availability

### Monitoring

Key metrics to monitor:
- `tasks_submitted_total`: Total tasks submitted
- `tasks_completed_total`: Total tasks completed
- `tasks_failed_total`: Total tasks failed
- `queue_size`: Current queue size
- `worker_count`: Active worker count
- `task_duration_seconds`: Task processing time

## Deployment

### Docker

```bash
# Build image
docker build -t task-scheduler .

# Run container
docker run -p 9090:9090 -p 8081:8081 task-scheduler
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: task-scheduler
spec:
  replicas: 3
  selector:
    matchLabels:
      app: task-scheduler
  template:
    metadata:
      labels:
        app: task-scheduler
    spec:
      containers:
      - name: scheduler
        image: task-scheduler:latest
        ports:
        - containerPort: 9090
        - containerPort: 8081
        env:
        - name: NODE_ID
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
```

## Project Structure

```
task-scheduler/
├── cmd/scheduler/          # Main application
├── internal/
│   ├── grpc/              # gRPC API layer
│   ├── cluster/           # Leader election & coordination
│   ├── queue/             # Priority queue
│   ├── scheduler/         # Main scheduler logic
│   ├── storage/           # Storage abstraction
│   ├── types/             # Data types
│   └── worker/            # Worker pool
├── proto/                 # Protocol buffer definitions
├── docker-compose.yml     # Local development
├── Dockerfile            # Container build
└── README.md             # This file
```

## Development

### Available Make Commands

```bash
make build              # Build the scheduler binary
make test               # Run unit tests
make test-integration   # Run integration tests
make benchmark          # Run benchmarks
make clean              # Clean build artifacts
make docker-build       # Build Docker image
make docker-run         # Run with Docker Compose
make docker-stop        # Stop Docker Compose
make lint               # Run linter
make fmt                # Format code
make proto              # Generate protobuf code
```

### Adding New Features

1. **Task Dependencies**: Implement task dependency graph
2. **Timeouts**: Add configurable task timeouts
3. **Web UI**: Create dashboard for monitoring
4. **Autoscaling**: Implement horizontal autoscaling
5. **Task Retries**: Add retry logic for failed tasks

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details.