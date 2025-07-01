# ADR 001: Hexagonal Architecture Pattern

## Status

Accepted

## Context

The Event Streaming Gateway needs to process high-volume events with low latency while maintaining clean separation of concerns. The system must be testable, maintainable, and independent of external frameworks.

Several architectural patterns were considered:
- Monolithic architecture
- Microservices architecture
- Layered (N-tier) architecture
- Hexagonal (Ports & Adapters) architecture

## Decision

We adopt the **Hexagonal Architecture** pattern for the Event Streaming Gateway.

## Rationale

### Separation of Concerns

Hexagonal architecture provides clear separation between:
- **Core Domain Logic**: Independent of any framework or transport mechanism
- **Ports**: Define contracts for communication with external systems
- **Adapters**: Implement specific technology bindings (gRPC, HTTP, Kafka, etc.)

This separation ensures that the core event processing logic is unaffected by changes to:
- Transport protocols (can add REST API without touching core logic)
- Message brokers (can switch from Kafka to RabbitMQ with minimal changes)
- Storage backends (can swap persistence layers easily)

### Testability

With clear boundaries:
- Core domain logic can be tested in isolation without external dependencies
- Adapters can be tested with mock ports
- Integration tests can use in-memory implementations of ports
- No need for complex mocking frameworks for core business logic

### Framework Independence

The core business logic depends on abstractions (interfaces), not concrete implementations:
- No vendor lock-in to specific frameworks
- Easier to upgrade or switch dependencies
- Core logic remains stable while technology choices evolve

### Technology Flexibility

Ports define flexible integration points:
- **Input Ports**: Accept events via gRPC, HTTP REST, or message queues
- **Output Ports**: Emit events to Kafka, RabbitMQ, databases, or webhooks
- New adapters can be added without modifying core logic or existing adapters

### Performance Considerations

Hexagonal architecture aligns well with performance requirements:
- Direct function calls in core logic (no framework overhead)
- Adapters can implement efficient, protocol-specific optimizations
- Clear resource management and lifecycle boundaries
- Easy to profile and optimize performance-critical paths

## Implementation

### Directory Structure

```
gateway/
├── cmd/
│   └── gateway/              # Application entry point
├── internal/
│   ├── api/                  # API handlers (input adapters)
│   │   └── grpc/             # gRPC API handlers
│   ├── application/          # Application layer
│   │   ├── dto/              # Data transfer objects
│   │   ├── port/             # Port interfaces (contracts)
│   │   │   ├── input.go      # Input ports
│   │   │   └── output.go     # Output ports
│   │   └── service/          # Application services
│   ├── domain/               # Domain layer (core business logic)
│   │   ├── entity/           # Domain entities
│   │   ├── valueobject/      # Value objects
│   │   └── processor/        # Domain processors
│   └── infrastructure/       # Infrastructure layer (output adapters)
│       ├── kafka/            # Kafka producer/consumer
│       ├── metrics/          # Prometheus metrics
│       ├── config/           # Configuration management
│       ├── logging/          # Logging utilities
│       ├── auth/             # Authentication/authorization
│       └── ...               # Other infrastructure components
└── pkg/                       # Reusable packages
```

### Port Definitions

```go
// Input ports define how events enter the system
type EventReceiver interface {
    ReceiveEvent(ctx context.Context, event *Event) error
}

// Output ports define how events leave the system
type EventPublisher interface {
    PublishEvent(ctx context.Context, event *Event) error
}
```

### Adapter Examples

- **gRPC API Adapter**: Implements Protocol Buffers-based communication (`internal/api/grpc/`)
- **Kafka Adapter**: Reads from Kafka topics and writes processed events (`internal/infrastructure/kafka/`)
- **Metrics Adapter**: Exports Prometheus metrics (`internal/infrastructure/metrics/`)
- **Schema Adapter**: Validates events against schemas (`internal/infrastructure/schema/`)
- **Auth Adapter**: Authenticates and authorizes requests (`internal/infrastructure/auth/`)

## Consequences

### Advantages

- Core logic is framework-agnostic and highly testable
- Easy to add new input/output protocols
- Clear architectural boundaries improve code navigation
- Performance-critical paths are unencumbered by framework overhead
- Reduces coupling between components
- Facilitates team collaboration with clear responsibilities

### Disadvantages

- Requires initial infrastructure setup for ports and adapters
- Developers must understand the port/adapter pattern
- May have minimal overhead from interface indirection (negligible in Go)
- Risk of over-engineering for simple use cases (mitigated by starting small)

## Related ADRs

- ADR 002: Kafka Message Broker Selection
- ADR 003: gRPC API Protocol

## References

- Cockburn, A. (2005). "Hexagonal Architecture"
- Newman, S. (2015). "Building Microservices" - Chapter on Evolutionary Architecture
- Go Project Layout: https://github.com/golang-standards/project-layout
