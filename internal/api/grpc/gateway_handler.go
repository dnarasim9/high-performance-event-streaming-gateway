package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	gatewayv1 "github.com/dheemanth-hn/event-streaming-gateway/gen/go/gateway/v1"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/port"
)

// GatewayHandler implements the GatewayService gRPC server interface.
type GatewayHandler struct {
	gatewayv1.UnimplementedGatewayServiceServer
	consumerManagementUseCase port.ConsumerManagementUseCase
	healthChecker             HealthChecker
	metricsCollector          port.MetricsCollector
	logger                    *slog.Logger
}

// HealthChecker defines the interface for checking health status.
type HealthChecker interface {
	GetStatus(ctx context.Context) (*HealthStatus, error)
}

// HealthStatus represents the health status of a service.
type HealthStatus struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// NewGatewayHandler creates a new GatewayHandler with the specified dependencies.
func NewGatewayHandler(
	consumerManagementUseCase port.ConsumerManagementUseCase,
	healthChecker HealthChecker,
	metricsCollector port.MetricsCollector,
	logger *slog.Logger,
) *GatewayHandler {
	return &GatewayHandler{
		consumerManagementUseCase: consumerManagementUseCase,
		healthChecker:             healthChecker,
		metricsCollector:          metricsCollector,
		logger:                    logger,
	}
}

// GetHealth returns the health status of the gateway.
func (h *GatewayHandler) GetHealth(ctx context.Context, _ *emptypb.Empty) (*gatewayv1.HealthStatus, error) {
	status, err := h.healthChecker.GetStatus(ctx)
	if err != nil {
		h.logger.Error("failed to get health status", "error", err)
		return nil, grpcError(codes.Internal, "failed to get health status")
	}

	// Convert status string to proto enum
	statusEnum := gatewayv1.HealthStatus_STATUS_UNHEALTHY
	switch status.Status {
	case "HEALTHY":
		statusEnum = gatewayv1.HealthStatus_STATUS_HEALTHY
	case "DEGRADED":
		statusEnum = gatewayv1.HealthStatus_STATUS_DEGRADED
	case "UNHEALTHY":
		statusEnum = gatewayv1.HealthStatus_STATUS_UNHEALTHY
	}

	return &gatewayv1.HealthStatus{
		Status:     statusEnum,
		Message:    status.Message,
		Components: make(map[string]string),
	}, nil
}

// GetMetrics returns current metrics for the gateway.
func (h *GatewayHandler) GetMetrics(ctx context.Context, _ *emptypb.Empty) (*gatewayv1.GatewayMetrics, error) {
	// TODO: Implement metrics retrieval from MetricsCollector
	// For now, return a placeholder response with correct field names
	return &gatewayv1.GatewayMetrics{
		TotalEventsIngested:  0,
		TotalEventsProcessed: 0,
		ActiveSubscriptions:  0,
		ActiveConnections:    0,
		AvgLatencyMs:         0.0,
		ErrorCount:           0,
		EventsBySource:       make(map[string]int64),
		EventsByType:         make(map[string]int64),
	}, nil
}

// ListConsumers returns a list of registered consumers.
func (h *GatewayHandler) ListConsumers(ctx context.Context, req *gatewayv1.ListConsumersRequest) (*gatewayv1.ListConsumersResponse, error) {
	if req == nil {
		return nil, grpcError(codes.InvalidArgument, "request is required")
	}

	// Call use case - pass empty string for group ID as it's not in the proto
	consumers, err := h.consumerManagementUseCase.List(ctx, "")
	if err != nil {
		h.logger.Error("failed to list consumers", "error", err)
		return nil, grpcError(codes.Internal, "failed to list consumers")
	}

	// Convert DTOs to proto messages
	protoConsumers := make([]*gatewayv1.Consumer, len(consumers))
	for i, c := range consumers {
		protoConsumers[i] = &gatewayv1.Consumer{
			Id:     c.ID,
			Name:   c.Name,
			Active: c.Status == "Active",
		}
	}

	return &gatewayv1.ListConsumersResponse{
		Consumers:  protoConsumers,
		TotalCount: int32(len(protoConsumers)), //nolint:gosec // safe conversion, list size will never overflow int32
	}, nil
}

// RegisterConsumer registers a new consumer with the gateway.
func (h *GatewayHandler) RegisterConsumer(ctx context.Context, req *gatewayv1.RegisterConsumerRequest) (*gatewayv1.RegisterConsumerResponse, error) {
	if req == nil || req.Consumer == nil || req.Consumer.Name == "" {
		return nil, grpcError(codes.InvalidArgument, "consumer and consumer name are required")
	}

	// Convert proto request to DTO
	dtoReq := consumerProtoToDTO(req)
	if dtoReq == nil {
		return nil, grpcError(codes.InvalidArgument, "failed to convert request")
	}

	// Call use case
	consumerID, err := h.consumerManagementUseCase.Register(ctx, dtoReq)
	if err != nil {
		h.logger.Error("failed to register consumer", "error", err, "name", req.Consumer.Name)
		return nil, grpcError(codes.Internal, "failed to register consumer")
	}

	return &gatewayv1.RegisterConsumerResponse{
		ConsumerId: consumerID,
		Success:    true,
	}, nil
}

// DeregisterConsumer removes a consumer from the gateway.
func (h *GatewayHandler) DeregisterConsumer(ctx context.Context, req *gatewayv1.DeregisterConsumerRequest) (*gatewayv1.DeregisterConsumerResponse, error) {
	if req == nil || req.ConsumerId == "" {
		return nil, grpcError(codes.InvalidArgument, "consumer ID is required")
	}

	// Call use case
	err := h.consumerManagementUseCase.Deregister(ctx, req.ConsumerId)
	if err != nil {
		h.logger.Error("failed to deregister consumer", "error", err, "consumer_id", req.ConsumerId)
		return nil, grpcError(codes.Internal, "failed to deregister consumer")
	}

	return &gatewayv1.DeregisterConsumerResponse{}, nil
}

// grpcError creates a gRPC status error with proper formatting.
func grpcError(code codes.Code, message string) error {
	return status.Error(code, message)
}
