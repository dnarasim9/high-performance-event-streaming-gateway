# ADR 002: Kafka as Message Broker

## Status

Accepted

## Context

The Event Streaming Gateway must process high-volume, continuous event streams with guarantees around message delivery and ordering. The system needs to support multiple consumers, replay of historical events, and integration with existing systems.

Multiple message broker options were evaluated:
- Apache Kafka
- RabbitMQ
- NATS
- Redis Streams
- AWS Kinesis
- Google Pub/Sub

## Decision

We select **Apache Kafka** as the primary message broker for event streaming.

## Rationale

### Throughput and Performance

Kafka is purpose-built for high-throughput streaming:
- Handles millions of messages per second
- Supports batching and compression for efficient network utilization
- Sequential writes to disk enable high performance at scale
- Allows tuning for latency vs. throughput trade-offs

Our performance targets are achievable with Kafka's architecture:
- Sub-second latencies for individual events
- Hundreds of thousands to millions of events per second per instance
- Horizontal scaling through partitioning

### Durability and Reliability

Kafka provides strong delivery guarantees:
- **Replication**: Multiple replicas across brokers prevent data loss
- **Persistence**: Events are durably stored on disk
- **Acknowledgment Modes**: Application controls trade-off between speed and safety
  - `acks=1`: Fast, suitable for non-critical events
  - `acks=all`: Durable, suitable for critical transactions

### Event Ordering and Partitioning

Kafka maintains message ordering per partition:
- Events for the same entity can be routed to the same partition
- Guarantees ordering for causally-related events
- Enables stateful processing (e.g., order of transactions)

Partitioning also enables:
- Parallel processing across multiple consumers
- Horizontal scaling
- Per-partition monitoring and debugging

### Consumer Flexibility

Kafka's consumer group model enables multiple consumption patterns:
- **Multiple Consumers**: Different systems can consume the same stream independently
- **Event Replay**: Consumers can reset to earlier offsets to reprocess events
- **Late Arrivals**: New consumers can subscribe and catch up from any point
- **Load Balancing**: Consumer groups automatically balance partitions across instances

This flexibility is critical for:
- Running multiple event processors
- A/B testing with different processing logic
- Debugging issues by replaying events
- Adding new consumers without affecting existing ones

### Operational Maturity

Kafka has mature ecosystem and tooling:
- Confluent Platform provides enterprise features
- Strong community and extensive documentation
- Multiple Go client libraries (Shopify/sarama, confluentinc/confluent-kafka-go)
- Well-understood deployment patterns and best practices
- Kafka Connect for integrations with external systems
- Schema Registry for data governance (optional)

### Integration Ecosystem

Kafka integrates with many systems:
- Spark, Flink for stream processing
- Data warehouses (Snowflake, BigQuery, Redshift)
- Analytics platforms
- Event processing frameworks
- Monitoring and observability tools

This enables future expansion without rearchitecting core streaming layer.

## Implementation

### Configuration

```go
ProducerConfig := sarama.NewConfig()
ProducerConfig.Producer.RequiredAcks = sarama.WaitForAll
ProducerConfig.Producer.Return.Successes = true
ProducerConfig.Producer.Compression = sarama.CompressionSnappy
ProducerConfig.Producer.Flush.Bytes = 16384
ProducerConfig.Producer.Flush.Messages = 100
ProducerConfig.Producer.Flush.MaxMessages = 1000
```

### Topic Design

Topics are created with appropriate settings:
- **Partitions**: Based on throughput and consumer parallelism
  - General rule: 10 partitions per GB/sec of target throughput
- **Replication Factor**: Typically 3 for production
  - 2 minimum for HA
  - More than 3 provides marginal benefits and increases storage
- **Retention**: Depends on use case
  - Events: 7 days
  - Raw data: 30 days
  - Audit logs: 90+ days

### Consumer Group Strategy

Different components use separate consumer groups:
- **Event Processor**: Consumes all events, transforms and filters
- **Analytics**: Consumes for BI and reporting
- **Audit**: Consumes for compliance and debugging
- **Backup**: Consumes for long-term storage

Each group tracks offsets independently, enabling independent scaling and failure isolation.

## Deployment Considerations

### Production Deployment

- Minimum 3 broker cluster for HA
- Separate ZooKeeper ensemble (3+ nodes) or KRaft mode (Kafka 3.3+)
- Proper monitoring of broker health, replication lag, and consumer lag
- Regular backup and disaster recovery procedures

### Docker/Kubernetes

- Confluent Platform images for simplified deployment
- StatefulSet for consistent broker identity
- PersistentVolumes for durable storage
- Proper resource requests and limits

### High Availability

- Distribute brokers across availability zones
- Configure min.insync.replicas = 2 for durability
- Monitor replication lag and unclean leader election
- Implement consumer group monitoring

## Consequences

### Advantages

- Proven at massive scale (LinkedIn, Uber, Netflix, etc.)
- Excellent for event streaming and stream processing
- Supports multiple consumption patterns
- Event replay capability crucial for debugging
- Strong ordering guarantees per partition
- Mature ecosystem and tooling
- Excellent operational characteristics

### Disadvantages

- Operational complexity (ZooKeeper/KRaft coordination)
- Resource overhead compared to lighter alternatives (RabbitMQ, NATS)
- Overkill for simple point-to-point messaging
- JVM dependency in standard deployments (use Confluent for Go alternatives)
- Learning curve for optimal configuration

## Trade-offs Made

- **vs. RabbitMQ**: Chose Kafka for replay and multiple consumption patterns, sacrificing some operational simplicity
- **vs. NATS**: Chose Kafka for proven scalability and ecosystem despite NATS being simpler
- **vs. Cloud Services**: Chose self-hosted Kafka for control and cost at expected volumes

## Monitoring and Alerting

Key metrics to monitor:
- Consumer lag (how far behind producers each consumer is)
- Broker disk usage and replication lag
- Producer throughput and error rates
- ZooKeeper quorum status
- Network I/O and bandwidth utilization

## Future Considerations

- Migration to Kafka KRaft (removing ZooKeeper dependency)
- Integration with Schema Registry for data validation
- Kafka Streams for complex event processing
- Tiered storage for cost optimization of historical data

## Related ADRs

- ADR 001: Hexagonal Architecture
- ADR 003: gRPC API Protocol

## References

- Kleppmann, M. (2017). "Designing Data-Intensive Applications" - Chapter on Streaming
- Kafka Documentation: https://kafka.apache.org/documentation/
- Confluent Blog: Event Streaming Platform Concepts
- LinkedIn Engineering: Real-time analytics at massive scale
