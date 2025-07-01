# Deployment and Testing Guide

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Local Development](#local-development)
3. [Testing](#testing)
4. [Docker Deployment](#docker-deployment)
5. [Kubernetes Deployment](#kubernetes-deployment)
6. [CI/CD Pipeline](#cicd-pipeline)
7. [Configuration Reference](#configuration-reference)
8. [Health Checks and Monitoring](#health-checks-and-monitoring)
9. [Troubleshooting](#troubleshooting)
10. [Production Checklist](#production-checklist)

---

## Prerequisites

### Required Tools

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.24+ | Build and test |
| Docker | 24+ | Container builds |
| Docker Compose | 2.0+ | Local stack orchestration |
| kubectl | 1.28+ | Kubernetes deployments |
| protoc | 25+ | Protobuf code generation |
| golangci-lint | 1.64+ | Static analysis |

### Install Tools

```bash
# Install all Go tools
make install-tools

# Verify Go installation
go version  # Should output go1.24+
```

---

## Local Development

### Build from Source

```bash
# Download dependencies
make deps

# Build for development (current OS/arch, with debug symbols)
make build-dev

# Build for production (linux/amd64, stripped binary)
make build
```

### Run Locally

The gateway requires a running Kafka cluster. Start one with Docker Compose:

```bash
# Start Kafka + Zookeeper + Prometheus
make docker-compose-up

# Run the gateway binary
KAFKA_BROKERS=localhost:9092 \
GRPC_PORT=50051 \
HTTP_PORT=8080 \
LOG_LEVEL=info \
./gateway
```

### Verify the Gateway is Running

```bash
# Health check
curl http://localhost:8080/health

# Readiness check
curl http://localhost:8080/ready

# Prometheus metrics
curl http://localhost:8080/metrics
```

### Send a Test Event via gRPC

```bash
# Install grpcurl if not present
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# List available services
grpcurl -plaintext localhost:50051 list

# Ingest a single event
grpcurl -plaintext -d '{
  "event": {
    "source": "test-client",
    "type": "test.event.created",
    "subject": "/tests/1",
    "data": "eyJtZXNzYWdlIjogImhlbGxvIn0=",
    "data_content_type": "application/json"
  }
}' localhost:50051 event.v1.EventService/IngestEvent
```

---

## Testing

### Test Categories

| Category | Command | Description |
|----------|---------|-------------|
| All tests | `make test` | Unit + integration with race detector |
| Short tests | `make test-short` | Fast subset only |
| Benchmarks | `make bench` | Performance benchmarks |
| Coverage | `make coverage` | Tests with HTML coverage report |
| Lint | `make lint` | Static analysis |
| Vet | `make vet` | Go vet checks |

### Running Tests

```bash
# Run all tests with race detection
make test

# Run tests with coverage report
make coverage
# Opens coverage.html in your browser

# Run benchmarks
make bench

# Run specific package tests
go test -v -race ./internal/application/service/...
go test -v -race ./internal/domain/...
go test -v -race ./pkg/circuitbreaker/...
```

### Test Coverage by Package

| Package | Coverage | Description |
|---------|----------|-------------|
| `pkg/retry` | 98.0% | Exponential backoff with jitter |
| `pkg/middleware` | 97.4% | gRPC correlation ID interceptors |
| `pkg/circuitbreaker` | 97.0% | Circuit breaker state machine |
| `domain/valueobject` | 92.7% | Event filters, transformations |
| `application/service` | 90.1% | Core business logic |
| `infrastructure/replay` | 85.8% | In-memory event store |
| `domain/entity` | 75.0% | Event, Consumer, Schema entities |
| `infrastructure/transformer` | 73.9% | JQ, CEL, Template engines |
| `infrastructure/health` | 72.5% | Health checker |
| `infrastructure/ratelimiter` | 71.6% | Token bucket rate limiter |
| `infrastructure/schema` | 67.6% | JSON Schema validation |
| `infrastructure/auth` | 61.5% | JWT + RBAC |

### Running Benchmarks

```bash
# All benchmarks
go test ./test/benchmark/... -bench=. -benchmem -benchtime=1s

# Specific benchmark
go test ./test/benchmark/... -bench=BenchmarkEventIngestion -benchmem -benchtime=3s

# Profile CPU
go test ./test/benchmark/... -bench=BenchmarkEventIngestion -cpuprofile=cpu.prof
go tool pprof cpu.prof

# Profile memory
go test ./test/benchmark/... -bench=BenchmarkEventIngestion -memprofile=mem.prof
go tool pprof mem.prof
```

### Key Benchmark Results

| Operation | Throughput | Latency | Allocations |
|-----------|-----------|---------|-------------|
| Event creation | ~6.3M ops/sec | 157 ns/op | 3 allocs/op |
| Event ingestion (full pipeline) | ~796K ops/sec | 1.26 µs/op | 21 allocs/op |
| Batch ingestion (10 events) | ~191K ops/sec | 5.24 µs/op | 97 allocs/op |
| Rate limiter (Allow) | ~14.3M ops/sec | 70 ns/op | 0 allocs/op |
| Circuit breaker (Execute) | ~214M ops/sec | 4.7 ns/op | 0 allocs/op |
| Event.Data() access | ~3.7B ops/sec | 0.27 ns/op | 0 allocs/op |
| Event store read by ID | ~186M ops/sec | 5.4 ns/op | 0 allocs/op |

---

## Docker Deployment

### Build Docker Image

```bash
# Build with default tag
make docker-build

# Build with custom registry and tag
DOCKER_REGISTRY=ghcr.io/dheemanth-hn DOCKER_TAG=v1.0.0 make docker-build

# Push to registry
DOCKER_REGISTRY=ghcr.io/dheemanth-hn DOCKER_TAG=v1.0.0 make docker-push
```

### Docker Compose (Full Stack)

The Docker Compose stack includes Zookeeper, Kafka, the Event Gateway, and Prometheus.

```bash
# Start the full stack
make docker-compose-up

# View logs
make docker-compose-logs

# Stop the stack
make docker-compose-down
```

**Services and Ports:**

| Service | Port | URL |
|---------|------|-----|
| Event Gateway (gRPC) | 50051 | `grpc://localhost:50051` |
| Event Gateway (HTTP) | 8080 | `http://localhost:8080` |
| Kafka | 9092 | `kafka://localhost:9092` |
| Zookeeper | 2181 | `zk://localhost:2181` |
| Prometheus | 9090 | `http://localhost:9090` |

### Standalone Docker Run

```bash
docker run -d \
  --name event-gateway \
  -p 50051:50051 \
  -p 8080:8080 \
  -e KAFKA_BROKERS=kafka:9092 \
  -e GRPC_PORT=50051 \
  -e HTTP_PORT=8080 \
  -e LOG_LEVEL=info \
  -e AUTH_ENABLED=true \
  -e JWT_SECRET=your-production-secret \
  -e RATE_LIMIT_RPS=10000 \
  event-gateway:latest
```

---

## Kubernetes Deployment

### Deploy to Kubernetes

```bash
# Deploy all resources (configmap, deployment, service, HPA)
make k8s-deploy

# Deploy to a specific namespace
NAMESPACE=production make k8s-deploy

# Check deployment status
make k8s-status

# View logs
make k8s-logs

# Open shell in pod
make k8s-shell
```

### Kubernetes Manifests

Located in `deployments/kubernetes/`:

| File | Resource | Description |
|------|----------|-------------|
| `configmap.yaml` | ConfigMap | Environment variables and configuration |
| `deployment.yaml` | Deployment | 3 replicas, rolling update, resource limits |
| `service.yaml` | Service | ClusterIP with gRPC and HTTP ports |
| `hpa.yaml` | HorizontalPodAutoscaler | CPU-based autoscaling (3-10 replicas) |

### Deployment Topology

```
                    ┌──────────────────────┐
                    │   Kubernetes Cluster  │
                    │                       │
                    │  ┌─────────────────┐  │
                    │  │    Service       │  │
          ─────────┼──│  event-gateway   │  │
          gRPC:50051│  │  (ClusterIP)    │  │
          HTTP:8080 │  └────────┬────────┘  │
                    │           │            │
                    │     ┌─────┼─────┐      │
                    │     ▼     ▼     ▼      │
                    │  ┌─────┐┌─────┐┌─────┐ │
                    │  │Pod 1││Pod 2││Pod 3│ │
                    │  └──┬──┘└──┬──┘└──┬──┘ │
                    │     │      │      │    │
                    │     └──────┼──────┘    │
                    │            ▼           │
                    │     ┌────────────┐     │
                    │     │   Kafka    │     │
                    │     │  Cluster   │     │
                    │     └────────────┘     │
                    └───────────────────────┘
```

### Resource Requests and Limits

| Resource | Request | Limit |
|----------|---------|-------|
| CPU | 250m | 1000m |
| Memory | 256Mi | 512Mi |

### Security Configuration

The Kubernetes deployment includes:

- Runs as non-root user (UID 1000)
- Read-only root filesystem
- All Linux capabilities dropped
- Seccomp profile: RuntimeDefault
- Pod anti-affinity for high availability
- 30-second graceful termination period

### Helm Chart

A Helm chart is available at `deployments/helm/event-gateway/` for templated deployments:

```bash
# Install with Helm
helm install event-gateway deployments/helm/event-gateway/ \
  --namespace production \
  --set kafka.brokers=kafka.production.svc:9092 \
  --set auth.enabled=true \
  --set auth.jwtSecret=your-secret

# Upgrade
helm upgrade event-gateway deployments/helm/event-gateway/ \
  --namespace production \
  --set image.tag=v1.1.0

# Uninstall
helm uninstall event-gateway --namespace production
```

---

## CI/CD Pipeline

### GitHub Actions Workflows

Located in `.github/workflows/`:

#### ci.yml — Continuous Integration

Triggers on push to `main`, `develop`, and `feature/**` branches, and on PRs to `main` and `develop`.

**Pipeline Stages:**

```
┌──────┐    ┌──────┐    ┌───────┐    ┌────────┐    ┌──────────────┐
│ Lint │───▶│ Test │───▶│ Build │───▶│ Docker │───▶│ Security Scan│
└──────┘    └──────┘    └───────┘    └────────┘    └──────────────┘
```

| Job | Duration | Description |
|-----|----------|-------------|
| Lint | ~3 min | golangci-lint with full ruleset |
| Test | ~5 min | Unit + integration tests with Kafka sidecar, race detection, coverage upload to Codecov |
| Build | ~3 min | Production binary (linux/amd64, stripped) |
| Docker | ~5 min | Multi-stage Docker build, push to GHCR and Docker Hub |
| Security Scan | ~3 min | Trivy vulnerability scanner, results uploaded to GitHub Security tab |

#### release.yml — Release Pipeline

Triggers on version tags (`v*`). Builds multi-platform Docker images and creates GitHub releases.

### Required GitHub Secrets

| Secret | Required | Description |
|--------|----------|-------------|
| `DOCKER_USERNAME` | For Docker Hub push | Docker Hub username |
| `DOCKER_PASSWORD` | For Docker Hub push | Docker Hub access token |
| `DOCKER_REGISTRY` | For Docker Hub push | Docker Hub registry (e.g., `docker.io/username`) |

Note: `GITHUB_TOKEN` is automatically provided by GitHub Actions for GHCR pushes.

---

## Configuration Reference

All configuration is done via environment variables. No configuration files are needed.

### Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `GRPC_PORT` | `50051` | gRPC server port |
| `HTTP_PORT` | `8080` | HTTP server port (health, readiness, metrics) |
| `ENVIRONMENT` | `development` | Environment name |

### Kafka Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_BROKERS` | `localhost:9092` | Comma-separated Kafka broker addresses |
| `KAFKA_TOPIC` | `events` | Default event topic |
| `KAFKA_GROUP_ID` | `event-gateway` | Consumer group ID |
| `KAFKA_NUM_PARTITIONS` | `3` | Number of topic partitions |
| `KAFKA_REPLICATION_FACTOR` | `1` | Topic replication factor |
| `KAFKA_READ_TIMEOUT` | `10s` | Read timeout |
| `KAFKA_WRITE_TIMEOUT` | `10s` | Write timeout |
| `KAFKA_DIAL_TIMEOUT` | `10s` | Connection timeout |
| `KAFKA_COMMIT_INTERVAL` | `1s` | Offset commit interval |
| `KAFKA_SESSION_TIMEOUT` | `10s` | Consumer session timeout |

### Authentication Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `AUTH_ENABLED` | `false` | Enable JWT authentication |
| `JWT_SECRET` | `default-secret-change-in-production` | JWT signing secret (MUST change in production) |
| `JWT_ISSUER` | `event-gateway` | JWT token issuer |
| `JWT_TOKEN_EXPIRY` | `24h` | Token expiration duration |
| `JWT_ALLOWED_ISSUER` | `event-gateway` | Accepted JWT issuer |

### Rate Limiting Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `RATE_LIMIT_ENABLED` | `true` | Enable rate limiting |
| `RATE_LIMIT_RPS` | `1000` | Requests per second per client |
| `RATE_LIMIT_BURST` | `100` | Burst capacity |
| `RATE_LIMIT_CLEANUP_INTERVAL` | `5m` | Stale limiter cleanup interval |
| `RATE_LIMIT_STALE_THRESHOLD` | `10m` | Time before a limiter is considered stale |

### Logging Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `json` | Log format: `json`, `text` |
| `LOG_OUTPUT` | `stdout` | Log output: `stdout`, `stderr`, or file path |

### Metrics Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `METRICS_ENABLED` | `true` | Enable Prometheus metrics |
| `METRICS_PORT` | `9090` | Metrics port (served on HTTP server at `HTTP_PORT`) |
| `METRICS_PATH` | `/metrics` | Metrics endpoint path |

### Replay Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `REPLAY_MAX_DURATION` | `168h` | Maximum replay time window (7 days) |
| `REPLAY_BATCH_SIZE` | `1000` | Events per replay batch |
| `REPLAY_RATE_LIMIT` | `1000` | Max events/sec during replay |
| `REPLAY_MAX_RETENTION` | `168h` | Event retention period (7 days) |

### Schema Validation Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SCHEMA_VALIDATION_ENABLED` | `false` | Enable schema validation |
| `SCHEMA_REGISTRY_URL` | `http://localhost:8081` | Schema registry URL |
| `SCHEMA_CACHE_TTL` | `1h` | Schema cache duration |

---

## Health Checks and Monitoring

### HTTP Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Liveness probe — returns 200 if the process is alive |
| `/ready` | GET | Readiness probe — returns 200 if all dependencies are healthy |
| `/metrics` | GET | Prometheus metrics in OpenMetrics format |

### Prometheus Metrics

Key metrics exported:

| Metric | Type | Description |
|--------|------|-------------|
| `event_gateway_events_ingested_total` | Counter | Total events ingested |
| `event_gateway_events_published_total` | Counter | Total events published to Kafka |
| `event_gateway_ingestion_duration_ms` | Histogram | Ingestion latency in milliseconds |
| `event_gateway_publish_duration_ms` | Histogram | Kafka publish latency |
| `event_gateway_active_subscriptions` | Gauge | Current active subscriptions |
| `event_gateway_errors_total` | Counter | Total errors by operation and type |

### Prometheus Scrape Configuration

```yaml
scrape_configs:
  - job_name: 'event-gateway'
    scrape_interval: 15s
    static_configs:
      - targets: ['event-gateway:8080']
    metrics_path: '/metrics'
```

### Kubernetes Probes

The deployment configures:

- **Liveness probe**: `GET /health` — restarts the pod if the process is unresponsive (30s initial delay, 10s period, 3 failures)
- **Readiness probe**: `GET /ready` — removes the pod from service if dependencies are unhealthy (20s initial delay, 10s period, 3 failures)

---

## Troubleshooting

### Common Issues

**Gateway fails to start: "at least one Kafka broker must be configured"**
Ensure `KAFKA_BROKERS` is set and the Kafka cluster is reachable.

```bash
# Verify Kafka connectivity
kafkacat -b localhost:9092 -L
```

**Gateway starts but events aren't published**
Check the Kafka topic exists and the gateway has write permissions:

```bash
# List topics
kafka-topics --bootstrap-server localhost:9092 --list

# Create topic manually if needed
kafka-topics --bootstrap-server localhost:9092 \
  --create --topic events \
  --partitions 3 --replication-factor 1
```

**High latency on event ingestion**
Check rate limiter settings and Kafka write timeouts:

```bash
# Increase rate limit
RATE_LIMIT_RPS=10000 RATE_LIMIT_BURST=1000 ./gateway

# Check Prometheus metrics for bottlenecks
curl http://localhost:8080/metrics | grep duration
```

**JWT authentication errors**
Ensure `AUTH_ENABLED=true` and `JWT_SECRET` is set correctly:

```bash
# Generate a test token (for development)
# The gateway uses HS256 signing
AUTH_ENABLED=true JWT_SECRET=my-secret ./gateway
```

**Pod CrashLoopBackOff in Kubernetes**
Check pod logs and events:

```bash
kubectl logs -n default -l app=event-gateway --previous
kubectl describe pod -n default -l app=event-gateway
```

---

## Production Checklist

Before deploying to production, verify every item:

### Security

- [ ] `JWT_SECRET` set to a strong, unique secret (not the default)
- [ ] `AUTH_ENABLED=true` in production
- [ ] Kafka brokers use TLS/SASL authentication
- [ ] Network policies restrict pod-to-pod communication
- [ ] Secrets managed via Kubernetes Secrets or Vault, not env vars in ConfigMaps

### Reliability

- [ ] At least 3 replicas configured
- [ ] HPA configured with appropriate CPU/memory thresholds
- [ ] Pod disruption budgets set (`minAvailable: 2`)
- [ ] Kafka topic replication factor >= 3
- [ ] Graceful shutdown timeout matches `terminationGracePeriodSeconds`
- [ ] Circuit breaker thresholds tuned for your error rates
- [ ] Rate limiting configured per your SLA requirements

### Observability

- [ ] Prometheus scraping configured
- [ ] Alerting rules defined for error rate, latency p99, and consumer lag
- [ ] Log aggregation configured (Loki, ELK, or CloudWatch)
- [ ] Distributed tracing enabled (correlation IDs propagated)
- [ ] Dashboards created for key metrics

### Performance

- [ ] Kafka partitions match expected throughput (1 partition ≈ 100K events/sec)
- [ ] Resource limits tuned based on load testing
- [ ] Connection pool sizes appropriate for your Kafka cluster
- [ ] Event payload sizes within `MAX_MESSAGE_SIZE` limits

### Deployment

- [ ] Rolling update strategy with `maxUnavailable: 0`
- [ ] Docker image uses specific version tag (not `latest`)
- [ ] CI/CD pipeline runs lint, test, and security scan before deploy
- [ ] Rollback procedure documented and tested
- [ ] Canary or blue-green deployment strategy for major releases
