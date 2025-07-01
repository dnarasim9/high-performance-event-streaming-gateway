# ADR 003: gRPC as Primary API Protocol

## Status

Accepted

## Context

The Event Streaming Gateway requires a high-performance API for event ingestion and querying. The system must handle:
- High throughput (100k+ events/second)
- Low latency (sub-100ms)
- Efficient network utilization
- Multiplexing multiple concurrent requests
- Streaming for both client-to-server and server-to-client

Alternative protocols considered:
- REST/HTTP with JSON
- GraphQL
- WebSockets
- AMQP
- Custom binary protocol

## Decision

We adopt **gRPC** as the primary API protocol for the Event Streaming Gateway.

## Rationale

### Performance

gRPC delivers superior performance characteristics:

**Binary Serialization (Protocol Buffers)**
- Compact wire format: 3-10x smaller than JSON
- Fast serialization/deserialization in Go
- Minimal CPU overhead for encoding
- Enables higher throughput on same hardware

**HTTP/2 Multiplexing**
- Multiple streams over single TCP connection
- Reduced connection overhead
- Full-duplex communication
- Head-of-line blocking eliminated

**Results**
- 5-7x throughput improvement vs. REST/JSON
- Sub-millisecond per-message latency
- 80% reduction in bandwidth usage
- Better CPU efficiency

### Streaming Capabilities

gRPC provides first-class streaming support:
- **Server Streaming**: Gateway streams events to clients
- **Client Streaming**: Clients stream events to gateway
- **Bidirectional Streaming**: Simultaneous client/server streams
- **Flow Control**: Automatic backpressure handling

Enables use cases:
- Real-time event subscriptions
- Batch event ingestion
- Event processing pipelines
- Live metric streaming

### Strong Typing

Protocol Buffers provide strong typing:
- Compile-time validation of message schemas
- Clear API contracts
- Automatic code generation for multiple languages
- Version compatibility checking
- Self-documenting APIs

Benefits:
- Fewer runtime errors
- Better IDE support and autocomplete
- API evolution without breaking clients
- Clear contract between services

### Language Interoperability

gRPC supports 10+ languages with generated code:
- Go, Java, Python, Node.js, C#, C++, Ruby, PHP, Kotlin, Swift
- Clients can be implemented in any language
- Consistent behavior across language implementations
- Shared protobuf definitions across teams

### Ecosystem and Tooling

Rich gRPC ecosystem:
- **Protobuf**: Industry standard for structured data
- **gRPC Gateway**: Generate REST/JSON endpoints from same proto definitions
- **gRPC Web**: Browser clients via special proxy
- **Monitoring**: Built-in metadata for tracing (gRPC metadata)
- **Load Balancing**: Native support in modern load balancers
- **Go Libraries**: Excellent standard library support

### Production Readiness

gRPC is proven in production at scale:
- Used by Google, Netflix, Uber, Lyft
- CNCF project (Cloud Native Computing Foundation)
- Stable API and long-term support
- Extensive documentation and community

## Implementation

### Protocol Buffer Definition Example

```protobuf
syntax = "proto3";

package event.v1;

service EventService {
  // Ingest a single event
  rpc IngestEvent(IngestEventRequest) returns (IngestEventResponse);

  // Stream multiple events for batch ingestion
  rpc IngestEventStream(stream IngestEventRequest) returns (stream IngestEventResponse);

  // Subscribe to events matching filters
  rpc SubscribeEvents(SubscribeRequest) returns (stream Event);

  // Replay events from a time range
  rpc ReplayEvents(ReplayRequest) returns (stream Event);
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

message IngestEventRequest {
  Event event = 1;
}

message IngestEventResponse {
  string event_id = 1;
  bool success = 2;
  string error_message = 3;
}
```

### Server Implementation

```go
type EventHandler struct {
    eventService EventService
}

func (h *EventHandler) IngestEvent(
    ctx context.Context,
    req *pb.IngestEventRequest,
) (*pb.IngestEventResponse, error) {
    // Fast path with minimal overhead
    err := h.eventService.ProcessEvent(ctx, req.Event)
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    return &pb.IngestEventResponse{
        EventId: req.Event.Id,
        Success: true,
    }, nil
}

func (h *EventHandler) IngestEventStream(
    stream pb.EventService_IngestEventStreamServer,
) error {
    // Handle client streaming with automatic flow control
    for {
        req, err := stream.Recv()
        if err == io.EOF {
            return nil
        }
        if err != nil {
            return err
        }
        err = h.eventService.ProcessEvent(stream.Context(), req.Event)
        if err := stream.Send(&pb.IngestEventResponse{
            EventId: req.Event.Id,
            Success: err == nil,
            ErrorMessage: errMsg,
        }); err != nil {
            return err
        }
    }
}
```

### Configuration

```go
opts := []grpc.ServerOption{
    grpc.MaxConcurrentStreams(uint32(maxConnections)),
    grpc.ConnectionTimeout(10 * time.Second),
    grpc.KeepaliveParams(keepalive.ServerParameters{
        Time:    20 * time.Second,
        Timeout: 10 * time.Second,
    }),
    grpc.MaxRecvMsgSize(maxMessageSize),
    grpc.MaxSendMsgSize(maxMessageSize),
}
```

### Note on REST API

The Event Streaming Gateway provides gRPC as its primary API protocol. REST/JSON support is not currently implemented. The gateway focuses on gRPC's superior performance characteristics and streaming capabilities for event processing workloads.

Future versions may add optional gRPC Gateway support for REST compatibility, but the primary API will remain gRPC.

## Deployment

### Port Configuration

- **gRPC Port**: 50051 (standard gRPC port)
- **HTTP Server**: 8080 (Health checks and Prometheus metrics)
  - `/health` - Health status endpoint
  - `/ready` - Readiness check endpoint
  - `/metrics` - Prometheus metrics endpoint

### Load Balancing

gRPC requires special load balancer support:
- **Cloud**: GCP Cloud Load Balancing, AWS NLB with gRPC support
- **On-Prem**: Envoy, HAProxy, Nginx (1.13+)
- **Kubernetes**: Service Mesh (Istio) with native gRPC support

### Security

gRPC security features implemented:
- Built-in authentication via JWT tokens and metadata
- Rate limiting at application level
- RBAC (Role-Based Access Control) support

Future enhancements:
- TLS/mTLS support for transport security

## Client Examples

### Go Client Example

```go
conn, _ := grpc.Dial("localhost:50051")
defer conn.Close()

client := eventv1.NewEventServiceClient(conn)

event := &eventv1.Event{
    Id:        "evt-123",
    Type:      "order.created",
    Source:    "payment-service",
    Data:      payload,
    Timestamp: timestamppb.Now(),
}

response, _ := client.IngestEvent(ctx, &eventv1.IngestEventRequest{
    Event: event,
})

fmt.Printf("Event %s ingested: %v\n", response.EventId, response.Success)
```

### Python Client Example

```python
import grpc
from event.v1 import event_pb2, event_pb2_grpc
from google.protobuf.timestamp_pb2 import Timestamp

channel = grpc.aio.secure_channel('localhost:50051', credentials)
stub = event_pb2_grpc.EventServiceStub(channel)

event = event_pb2.Event(
    id="evt-123",
    type="order.created",
    source="payment-service",
    data=payload,
    timestamp=Timestamp.GetCurrentTime()
)

request = event_pb2.IngestEventRequest(event=event)
response = await stub.IngestEvent(request)

print(f"Event {response.event_id} ingested: {response.success}")
```

## Consequences

### Advantages

- Excellent performance characteristics (throughput, latency, bandwidth)
- First-class streaming support
- Strong typing and schema validation
- Multi-language support
- CNCF standard with excellent tooling
- Efficient resource utilization
- Native Kubernetes integration
- Automatic service documentation

### Disadvantages

- Steeper learning curve than REST/JSON
- Browser support requires gRPC Web or gateway
- Debugging slightly harder than REST (not HTTP in browser)
- Requires Protocol Buffer knowledge
- Not ideal for simple CRUD operations

## Trade-offs Made

- **vs. REST/HTTP**: Chose gRPC for 5-7x performance, accepting slightly more complexity
- **vs. WebSockets**: Chose gRPC for cleaner protocol and better load balancing support
- **vs. Custom Protocol**: Chose gRPC for ecosystem and language support

## Interoperability

The Event Streaming Gateway focuses on gRPC-first design for optimal performance and streaming capabilities. REST API support is not currently implemented but could be added in the future through gRPC Gateway if needed.

## Monitoring

gRPC provides rich monitoring options:
- Request/response metadata for tracing
- Stream state for flow control monitoring
- Built-in error codes and status details
- Integration with OpenTelemetry for observability

## Future Considerations

- gRPC-Web for browser-based clients
- Protocol Buffer reflection for debugging tools
- gRPC performance enhancements in protocol evolution
- Integration with service meshes for advanced traffic management

## Related ADRs

- ADR 001: Hexagonal Architecture
- ADR 002: Kafka Message Broker

## References

- gRPC Documentation: https://grpc.io/docs/
- Protocol Buffers: https://developers.google.com/protocol-buffers
- CNCF Project: https://www.cncf.io/
- "Building Microservices in Go" - Chapter on gRPC
- Google Cloud: gRPC Performance Best Practices
