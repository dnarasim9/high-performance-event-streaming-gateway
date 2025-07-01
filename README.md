# Event Streaming Gateway

A high-performance, production-ready event streaming gateway built in Go. Process millions of events per second with sub-millisecond latency, featuring gRPC APIs, Kafka integration, and cloud-native deployment patterns.

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [API Documentation](#api-documentation)
- [Configuration](#configuration)
- [Deployment](#deployment)
  - [Docker](#docker)
  - [Kubernetes](#kubernetes)
- [Performance](#performance)
- [Development](#development)
- [Contributing](#contributing)

## Features

### High Performance
- **Throughput**: ~796K events/second per instance (full pipeline)
- **Latency**: Sub-millisecond per-event processing
- **Bandwidth**: 3-10x more efficient than JSON (Protocol Buffers)
- **Scalability**: Linear horizontal scaling via Kubernetes or Docker Compose

### Reliable Event Processing
- **Durability**: Kafka-backed persistence with configurable replication
- **Ordering**: Per-partition message ordering guarantees
- **Delivery**: At-least-once processing semantics
- **Replay**: Full event replay capability for debugging and reprocessing

### Modern API Design
- **gRPC**: High-performance RPC with Protocol Buffers
- **Streaming**: Bidirectional streaming for efficient data transfer
- **Type-Safe**: Protocol Buffers provide strong typing and schema validation
- **Metadata**: Rich request/response metadata for tracing

### Production Ready
- **Observability**: Prometheus metrics, structured logging, distributed tracing
- **Health Checks**: Liveness and readiness probes
- **Security**: Rate limiting, circuit breakers (TLS support coming)
- **Deployment**: Docker, Kubernetes, cloud-native patterns

### Developer Friendly
- **Type Safety**: Protocol Buffers with strong typing
- **Hexagonal Architecture**: Clear separation of concerns
- **Testability**: Comprehensive testing utilities and examples
- **Documentation**: ADRs, API docs, deployment guides

## Architecture

### System Architecture

```
┌─────────────────────────────────────────────────────────┐
│                  Client Applications                     │
│         (gRPC Clients, Event Producers)                 │
└────────┬────────────────────────────────────┬───────────┘
         │                                    │
         │ gRPC                               │ gRPC Streaming
         ▼                                    ▼
┌──────────────────────────────────────────────────────────┐
│          Event Streaming Gateway (Cluster)               │
│  ┌────────────────────────────────────────────────────┐  │
│  │  gRPC Server        HTTP Server (Health/Metrics)  │  │
│  │  :50051             :8080                         │  │
│  └────────────────────────────────────────────────────┘  │
│  ┌────────────────────────────────────────────────────┐  │
│  │              Event Processor                        │  │
│  │   (Filtering, Transformation, Enrichment)          │  │
│  └────────────────────────────────────────────────────┘  │
│  ┌────────────────────────────────────────────────────┐  │
│  │          Kafka Producer/Consumer                   │  │
│  │     (Batch, Compression, Retry Logic)              │  │
│  └────────────────────────────────────────────────────┘  │
└────────┬───────────────────────────────────────────────┬─┘
         │                                               │
         │ Kafka Protocol                               │
         ▼                                               ▼
┌──────────────────────────────┐        ┌────────────────────┐
│     Apache Kafka Cluster     │        │ Prometheus Metrics │
│  (Replication, Persistence)  │        │                    │
└──────────────────────────────┘        └────────────────────┘
```

### Component Architecture (Hexagonal Pattern)

```
┌─────────────────────────────────────────┐
│         External World                   │
│  ┌──────────────┐  ┌──────────────┐    │
│  │ gRPC Clients │  │ Kafka Events │    │
│  └──────┬───────┘  └──────┬───────┘    │
└─────────┼──────────────────┼────────────┘
          │                  │
┌─────────▼──────────────────▼────────────┐
│            Input Adapters               │
│  ┌──────────────┐  ┌──────────────┐    │
│  │  gRPC Server │  │ Kafka Consumer   │
│  └──────┬───────┘  └──────┬───────┘    │
└─────────┼──────────────────┼────────────┘
          │                  │
          └────────┬─────────┘
                   │
┌──────────────────▼───────────────────────┐
│      Domain Layer                        │
│  ┌──────────────────────────────────┐   │
│  │     Event Processor Service      │   │
│  │  (Core Business Logic)            │   │
│  └──────────────────────────────────┘   │
└──────────────────┬──────────────────────┘
                   │
         ┌─────────┴──────────┐
         │                    │
┌────────▼──────────┐  ┌──────▼────────────┐
│  Output Adapters  │  │ Observability     │
│ ┌──────────────┐  │  │ ┌──────────────┐  │
│ │ Kafka Output │  │  │ │ Prometheus   │  │
│ │ Log Output   │  │  │ │ Tracing      │  │
│ └──────────────┘  │  │ └──────────────┘  │
└───────┬───────────┘  └────────┬──────────┘
        │                       │
        └───────────┬───────────┘
                    │
         ┌──────────▼──────────┐
         │  External Services  │
         │  (Kafka, Logs, etc) │
         └─────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.24+
- Docker & Docker Compose
- Kubernetes 1.24+ (for K8s deployment)
- Make

### Using Docker Compose

```bash
# Clone the repository
git clone https://github.com/your-org/event-streaming-gateway.git
cd event-streaming-gateway

# Start the full stack
make docker-compose-up

# Check services
curl http://localhost:8080/health
curl http://localhost:9090  # Prometheus
```

### Local Development

```bash
# Install dependencies
make deps

# Run tests
make test

# Build binary
make build

# Run gateway
KAFKA_BROKERS=localhost:9092 ./gateway
```

### Using Kubernetes

```bash
# Install to default namespace
make k8s-deploy

# Check deployment
kubectl get pods -l app=event-gateway

# Forward ports for testing
kubectl port-forward svc/event-gateway 50051:50051
```

## API Documentation

### gRPC Service Definition

The Event Streaming Gateway exposes a gRPC service with the following main operations:

#### IngestEvent
Ingest a single event for processing.

```protobuf
rpc IngestEvent(IngestEventRequest) returns (IngestEventResponse);
```

**Request:**
```protobuf
message IngestEventRequest {
  Event event = 1;
}

message Event {
  string id = 1;
  string source = 2;
  string type = 3;
  string subject = 4;
  bytes data = 5;
  string data_content_type = 6;
  google.protobuf.Timestamp timestamp = 8;
  map<string, string> metadata = 9;
  string partition_key = 12;
}
```

**Response:**
```protobuf
message IngestEventResponse {
  string event_id = 1;
  bool success = 2;
  string error_message = 3;
}
```

#### IngestEventStream
Stream multiple events to the gateway for batch ingestion.

```protobuf
rpc IngestEventStream(stream IngestEventRequest) returns (stream IngestEventResponse);
```

Enables batching multiple events in a single gRPC stream for maximum throughput.

#### SubscribeEvents
Subscribe to events matching specific criteria (server streaming).

```protobuf
rpc SubscribeEvents(SubscribeRequest) returns (stream Event);
```

**Request:**
```json
{
  "event_types": ["order.created", "order.updated"],
  "start_timestamp": 1692230400
}
```

### Health and Metrics

The gateway exposes HTTP endpoints for health checks and metrics:

```bash
# Health check
curl http://localhost:8080/health

# Readiness check
curl http://localhost:8080/ready

# Prometheus metrics
curl http://localhost:8080/metrics
```

### Error Handling

gRPC uses standard error codes:

- `OK (0)`: Success
- `INVALID_ARGUMENT (3)`: Invalid request format
- `RESOURCE_EXHAUSTED (8)`: Rate limit exceeded
- `INTERNAL (13)`: Server error
- `UNAVAILABLE (14)`: Service temporarily unavailable

Example error response:
```json
{
  "code": "INTERNAL",
  "message": "failed to process event",
  "details": [...]
}
```

## Configuration

### Environment Variables

```bash
# Kafka Configuration
KAFKA_BROKERS=kafka-1:9092,kafka-2:9092,kafka-3:9092  # Comma-separated list
KAFKA_TOPIC=events                          # Kafka topic name
KAFKA_GROUP_ID=event-gateway                # Consumer group ID
KAFKA_NUM_PARTITIONS=3                      # Topics created with N partitions (default: 3)
KAFKA_REPLICATION_FACTOR=1                  # Replication across N brokers (default: 1)
KAFKA_READ_TIMEOUT=10s                      # Read timeout duration
KAFKA_WRITE_TIMEOUT=10s                     # Write timeout duration
KAFKA_DIAL_TIMEOUT=10s                      # Connection dial timeout
KAFKA_COMMIT_INTERVAL=1s                    # Offset commit interval
KAFKA_SESSION_TIMEOUT=10s                   # Consumer session timeout

# Server Configuration
GRPC_PORT=50051                             # gRPC server port (default: 50051)
HTTP_PORT=8080                              # HTTP server port for health/metrics (default: 8080)
LOG_LEVEL=info                              # debug, info, warn, error (default: info)
LOG_FORMAT=json                             # json or text (default: json)
LOG_OUTPUT=stdout                           # stdout, stderr, or file path (default: stdout)
METRICS_ENABLED=true                        # Enable Prometheus metrics (default: true)
METRICS_PATH=/metrics                       # Metrics endpoint path (default: /metrics)

# Authentication
AUTH_ENABLED=false                          # Enable JWT authentication (default: false)
JWT_SECRET=your-secret-key                  # JWT signing secret
JWT_ISSUER=event-gateway                    # JWT issuer claim
JWT_TOKEN_EXPIRY=24h                        # Token expiry duration

# Rate Limiting
RATE_LIMIT_ENABLED=true                     # Enable rate limiting (default: true)
RATE_LIMIT_RPS=1000                         # Requests per second limit (default: 1000)
RATE_LIMIT_BURST=100                        # Burst capacity (default: 100)

# Replay Configuration
REPLAY_MAX_DURATION=168h                    # Maximum replay duration (default: 7 days)
REPLAY_BATCH_SIZE=1000                      # Batch size for replay (default: 1000)
REPLAY_RATE_LIMIT=1000                      # Replay rate limit in events/sec (default: 1000)

# Schema Validation
SCHEMA_VALIDATION_ENABLED=false              # Enable schema validation (default: false)
SCHEMA_REGISTRY_URL=http://localhost:8081  # Schema registry URL
SCHEMA_CACHE_TTL=1h                         # Schema cache TTL (default: 1 hour)

# Environment
ENVIRONMENT=development                     # development, staging, or production
```

### Configuration

The Event Streaming Gateway is configured entirely through environment variables. There is no YAML configuration file support. All configuration must be passed as environment variables when starting the application.

## Deployment

### Docker Deployment

#### Single Container

```bash
# Build image
docker build -t event-gateway:latest .

# Run container
docker run -d \
  -e KAFKA_BROKERS=kafka:9092 \
  -p 50051:50051 \
  -p 8080:8080 \
  event-gateway:latest
```

#### Full Stack with Docker Compose

```bash
# Start services
make docker-compose-up

# View logs
make docker-compose-logs

# Stop services
make docker-compose-down
```

Includes:
- Zookeeper
- Kafka (3 brokers)
- Event Gateway
- Prometheus
- Health checks and auto-restart

### Kubernetes Deployment

#### Install Resources

```bash
# Deploy to cluster
make k8s-deploy NAMESPACE=production

# Verify deployment
kubectl get all -n production -l app=event-gateway
kubectl logs -f -n production -l app=event-gateway
```

#### Verify Deployment

```bash
# Check pods
kubectl get pods -n production -l app=event-gateway

# Check services
kubectl get svc -n production -l app=event-gateway

# Check HPA status
kubectl get hpa -n production event-gateway

# View events
kubectl describe deployment -n production event-gateway
```

#### Access Services

```bash
# Port forward gRPC
kubectl port-forward -n production svc/event-gateway 50051:50051

# Port forward HTTP
kubectl port-forward -n production svc/event-gateway 8080:8080

# Open shell in pod
make k8s-shell NAMESPACE=production
```

#### Clean Up

```bash
# Delete resources
make k8s-delete NAMESPACE=production
```

### Production Checklist

- [ ] Configure proper Kafka cluster (minimum 3 brokers)
- [ ] Set `KAFKA_REQUIRED_ACKS=all` for durability
- [ ] Enable metrics and monitoring (Prometheus + Grafana)
- [ ] Configure TLS for Kafka and gRPC
- [ ] Set up proper logging aggregation (ELK, Loki, etc.)
- [ ] Configure HPA min/max replicas appropriately
- [ ] Set up alerts for critical metrics
- [ ] Test failover and disaster recovery procedures
- [ ] Implement proper RBAC in Kubernetes
- [ ] Configure network policies for security
- [ ] Set up proper backup and retention policies

## Performance

### Benchmarks

Performance tested on commodity hardware (4-core, 8GB RAM) with Go 1.24:

#### Event Processing Operations

| Operation | Throughput | Latency | Memory | Notes |
|-----------|-----------|---------|--------|-------|
| Event Creation | ~6.3M ops/sec | 157.6 ns/op | 336 B/op | Individual event object creation |
| Event Ingestion | ~796K ops/sec | 1,256 ns/op | 1,940 B/op | Full pipeline processing per event |
| Event Ingestion (Batch) | ~1.9M ops/sec | 5,237 ns/op | 7,219 B/op | 10 events per batch |
| JQ Transformation | ~44.9K ops/sec | 12,816 ns/op | 10,835 B/op | JQ-based event transformation |
| CEL Transformation | ~56.9K ops/sec | 10,443 ns/op | 6,465 B/op | CEL-based event transformation |
| Template Transformation | ~91.9K ops/sec | 6,589 ns/op | 4,067 B/op | Template-based transformation |

#### Reliability Components

| Component | Throughput | Latency | Memory | Notes |
|-----------|-----------|---------|--------|-------|
| Rate Limiter | ~14.3M ops/sec | 69.96 ns/op | Zero alloc | Concurrent-safe rate limiting |
| Circuit Breaker | ~214M ops/sec | 4.667 ns/op | Zero alloc | Fast circuit breaker checks |
| Rate Limiter (Concurrent) | ~7.4M ops/sec | 80.06 ns/op | Zero alloc | Concurrent access performance |

#### Event Storage

| Operation | Throughput | Latency | Memory | Notes |
|-----------|-----------|---------|--------|-------|
| Event Store Write | ~22.3M ops/sec | 44.92 ns/op | 47 B/op | In-memory event persistence |
| Event Store Read by ID | ~186M ops/sec | 5.365 ns/op | Zero alloc | Direct ID-based lookup |

#### Summary

**Key Highlights:**
- **Event creation**: ~6.3M events/sec with minimal allocation overhead
- **Event ingestion (full pipeline)**: ~796K events/sec
- **Batch ingestion**: ~1.9M events/sec (10 events/batch) - recommended for high throughput
- **Rate limiter**: ~14.3M ops/sec with zero allocations (extremely efficient)
- **Circuit breaker**: ~214M ops/sec with zero allocations (negligible overhead)
- **Event store**: ~22.3M writes/sec and ~186M reads/sec for in-memory operations

**Throughput Scaling with Kafka:**
- **10 partitions**: 796K events/sec × 3 instances = 2.4M/sec cluster throughput
- **30 partitions**: 796K events/sec × 10 instances = 7.96M/sec cluster throughput
- **100+ partitions**: Scales to tens of millions of events/sec

### Throughput Scaling

With Kafka partitioning and multiple instances:
- **10 partitions**: 796K events/sec × 3 instances = 2.4M/sec cluster throughput
- **30 partitions**: 796K events/sec × 10 instances = 7.96M/sec cluster throughput
- **100+ partitions**: Scales to tens of millions of events/sec

### Optimization Tips

1. **Batching**: Use `IngestEventStream` for batch ingestion (achieve ~1.9M ops/sec with 10 events per batch)
2. **Connection Pooling**: Reuse gRPC connections on client side
3. **Partition Key**: Use `partition_key` field for consistent routing of related events
4. **Concurrent Requests**: Leverage gRPC's HTTP/2 multiplexing for concurrent ingestion
5. **Event Size**: Keep individual event data reasonably sized (max message size: 20MB)

## Development

### Project Structure

```
event-streaming-gateway/
├── cmd/
│   └── gateway/
│       └── main.go              # Application entry point
├── internal/
│   ├── api/                     # API handlers (gRPC)
│   │   └── grpc/
│   ├── application/             # Application layer
│   │   ├── dto/                 # Data transfer objects
│   │   ├── port/                # Port interfaces
│   │   └── service/             # Application services
│   ├── domain/                  # Domain layer (business logic)
│   │   ├── entity/
│   │   ├── valueobject/
│   │   └── processor/
│   └── infrastructure/          # Infrastructure layer
│       ├── config/              # Configuration management
│       ├── kafka/               # Kafka producer/consumer
│       ├── metrics/             # Prometheus metrics
│       ├── auth/                # Authentication/authorization
│       ├── health/              # Health checks
│       ├── logging/             # Logging utilities
│       ├── repository/          # Data repositories
│       ├── ratelimiter/         # Rate limiting
│       ├── replay/              # Event replay
│       └── schema/              # Schema validation
├── pkg/                         # Public packages
│   └── middleware/              # gRPC middleware
├── proto/                       # Protocol Buffer definitions
│   ├── event/v1/
│   ├── gateway/v1/
│   └── schema/v1/
├── gen/                         # Generated protobuf code
├── docs/                        # Documentation
│   └── adr/                     # Architecture Decision Records
├── deployments/                 # Deployment configurations
│   ├── docker/
│   └── kubernetes/
├── scripts/                     # Build and deployment scripts
├── test/                        # Integration tests
├── Dockerfile                   # Container image definition
├── Makefile                     # Build automation
└── go.mod                       # Go module dependencies
```

### Common Tasks

```bash
# Format code
make fmt

# Lint code
make lint

# Run tests
make test

# Generate coverage report
make coverage

# Build binary
make build

# Clean build artifacts
make clean

# Generate protobuf code
make proto
```

### Testing

```bash
# Unit tests
make test

# With coverage
make coverage

# Benchmarks
make bench

# Short tests (quick feedback)
make test-short
```

### Adding New Features

1. Create feature branch: `git checkout -b feature/my-feature`
2. Update protobuf definitions if needed: `proto/event/v1/event.proto` or `proto/gateway/v1/gateway.proto`
3. Generate new proto code: `make proto`
4. Implement port interface in `internal/application/port/`
5. Add service implementation in `internal/application/service/`
6. Add API handler in `internal/api/grpc/`
7. Write tests in corresponding `*_test.go` files
8. Submit pull request with tests and documentation

## Contributing

### Code Standards

- Follow Go conventions and style guidelines
- Write clear, testable code
- Add tests for new functionality (target 80%+ coverage)
- Update documentation for user-facing changes
- Use descriptive commit messages

### Pull Request Process

1. Fork the repository
2. Create feature branch from `develop`
3. Make changes and add tests
4. Ensure all tests pass: `make test`
5. Run linter: `make lint`
6. Update documentation
7. Submit pull request with clear description

### Reporting Issues

- Use GitHub Issues for bug reports
- Include reproduction steps
- Provide relevant logs and configuration
- Suggest fixes if possible

## License

MIT License - see LICENSE file for details

## Support

- Documentation: See `/docs` directory
- Issues: GitHub Issues
- Architecture Decisions: See `/docs/adr/` directory
- API Specification: See Protocol Buffer definitions in `/api/proto/`

## Roadmap

- [ ] gRPC Web support for browser clients
- [ ] GraphQL API layer
- [ ] Event schema registry integration
- [ ] Stream processing pipelines
- [ ] Multi-region deployment guide
- [ ] Performance optimization guide
- [ ] Extended monitoring dashboards
- [ ] CLI tool for event management

## Acknowledgments

Built with:
- Go standard library
- Apache Kafka
- Protocol Buffers
- gRPC
- Prometheus
