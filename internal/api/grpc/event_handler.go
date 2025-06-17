package grpc

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	eventv1 "github.com/dheemanth-hn/event-streaming-gateway/gen/go/event/v1"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/dto"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/port"
)

// EventHandler implements the EventService gRPC server interface.
type EventHandler struct {
	eventv1.UnimplementedEventServiceServer
	ingestionUseCase    port.EventIngestionUseCase
	subscriptionUseCase port.EventSubscriptionUseCase
	replayUseCase       port.EventReplayUseCase
	logger              *slog.Logger
}

// NewEventHandler creates a new EventHandler with the specified use cases.
func NewEventHandler(
	ingestionUseCase port.EventIngestionUseCase,
	subscriptionUseCase port.EventSubscriptionUseCase,
	replayUseCase port.EventReplayUseCase,
	logger *slog.Logger,
) *EventHandler {
	return &EventHandler{
		ingestionUseCase:    ingestionUseCase,
		subscriptionUseCase: subscriptionUseCase,
		replayUseCase:       replayUseCase,
		logger:              logger,
	}
}

// IngestEvent handles a single event ingestion request.
func (h *EventHandler) IngestEvent(ctx context.Context, req *eventv1.IngestEventRequest) (*eventv1.IngestEventResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	// Convert proto request to DTO
	dtoReq := eventProtoToDTO(req)
	if dtoReq == nil {
		return nil, status.Error(codes.InvalidArgument, "failed to convert request")
	}

	// Call use case
	eventID, err := h.ingestionUseCase.IngestEvent(ctx, dtoReq)
	if err != nil {
		h.logger.Error("event ingestion failed", "error", err)
		return nil, status.Error(codes.Internal, "failed to ingest event")
	}

	// Build response - need to get the event to include timestamp
	// For now, return with current time since we don't have direct access to the event
	return &eventv1.IngestEventResponse{
		EventId: eventID,
	}, nil
}

// IngestEventStream handles bidirectional streaming of event ingestion.
// The client sends a stream of events and receives acknowledgments.
func (h *EventHandler) IngestEventStream(stream grpc.BidiStreamingServer[eventv1.IngestEventRequest, eventv1.IngestEventResponse]) error {
	batchRequests := make([]*eventv1.IngestEventRequest, 0, 100)
	const batchSize = 100

	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			// Client closed the stream, process remaining events
			if len(batchRequests) > 0 {
				if err := h.processBatch(stream.Context(), stream, batchRequests); err != nil {
					h.logger.Error("batch processing failed", "error", err)
					return status.Error(codes.Internal, "failed to process batch")
				}
			}
			return nil
		}
		if err != nil {
			h.logger.Error("failed to receive from stream", "error", err)
			return status.Error(codes.Internal, "failed to read from stream")
		}

		batchRequests = append(batchRequests, req)

		// Process batch when size is reached
		if len(batchRequests) >= batchSize {
			if err := h.processBatch(stream.Context(), stream, batchRequests); err != nil {
				h.logger.Error("batch processing failed", "error", err)
				return status.Error(codes.Internal, "failed to process batch")
			}
			batchRequests = nil
		}
	}
}

// processBatch processes a batch of event ingestion requests.
func (h *EventHandler) processBatch(ctx context.Context, stream grpc.BidiStreamingServer[eventv1.IngestEventRequest, eventv1.IngestEventResponse], requests []*eventv1.IngestEventRequest) error {
	// Convert proto requests to DTOs
	dtoRequests := make([]*dto.IngestEventRequest, len(requests))
	for i, req := range requests {
		dtoRequests[i] = eventProtoToDTO(req)
	}

	// Call use case batch method
	eventIDs, err := h.ingestionUseCase.IngestEventBatch(ctx, dtoRequests)
	if err != nil {
		return err
	}

	// Send responses back to client
	for _, eventID := range eventIDs {
		resp := &eventv1.IngestEventResponse{
			EventId: eventID,
		}
		if err := stream.Send(resp); err != nil {
			h.logger.Error("failed to send response", "error", err)
			return err
		}
	}

	return nil
}

// SubscribeEvents handles event subscription with server streaming.
// The client provides a subscription request and receives a stream of matching events.
func (h *EventHandler) SubscribeEvents(req *eventv1.SubscribeRequest, stream grpc.ServerStreamingServer[eventv1.Event]) error {
	if req == nil {
		return status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	// Convert proto request to DTO
	dtoReq := subscribeRequestProtoToDTO(req)
	if dtoReq == nil {
		return status.Error(codes.InvalidArgument, "converted request is nil")
	}

	// Call use case to get event channel
	eventChan, err := h.subscriptionUseCase.Subscribe(stream.Context(), dtoReq)
	if err != nil {
		h.logger.Error("subscription failed", "error", err)
		return status.Error(codes.Internal, "failed to subscribe to events")
	}

	// Stream events back to client
	for {
		select {
		case event := <-eventChan:
			if event == nil {
				// Channel closed, end stream
				return nil
			}

			// Convert entity to proto and send
			protoEvent := eventEntityToProto(event)
			if err := stream.Send(protoEvent); err != nil {
				h.logger.Error("failed to send event", "error", err)
				return status.Error(codes.Internal, "failed to send event")
			}

		case <-stream.Context().Done():
			// Client canceled the request
			return status.Error(codes.Canceled, "subscription canceled")
		}
	}
}

// ReplayEvents handles event replay with server streaming.
// The client provides a replay request with time range and receives a stream of matching events.
func (h *EventHandler) ReplayEvents(req *eventv1.ReplayRequest, stream grpc.ServerStreamingServer[eventv1.Event]) error {
	if req == nil {
		return status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	// Convert proto request to DTO
	dtoReq := replayRequestProtoToDTO(req)
	if dtoReq == nil {
		return status.Error(codes.InvalidArgument, "converted request is nil")
	}

	// Call use case to get event channel
	eventChan, err := h.replayUseCase.ReplayEvents(stream.Context(), dtoReq)
	if err != nil {
		h.logger.Error("replay failed", "error", err)
		return status.Error(codes.Internal, "failed to replay events")
	}

	// Stream events back to client
	for {
		select {
		case event := <-eventChan:
			if event == nil {
				// Channel closed, end stream
				return nil
			}

			// Convert entity to proto and send
			protoEvent := eventEntityToProto(event)
			if err := stream.Send(protoEvent); err != nil {
				h.logger.Error("failed to send event", "error", err)
				return status.Error(codes.Internal, "failed to send event")
			}

		case <-stream.Context().Done():
			// Client canceled the request
			return status.Error(codes.Canceled, "replay canceled")
		}
	}
}

// GetEvent retrieves a single event by its ID.
func (h *EventHandler) GetEvent(ctx context.Context, req *eventv1.GetEventRequest) (*eventv1.GetEventResponse, error) {
	if req == nil || req.EventId == "" {
		return nil, status.Error(codes.InvalidArgument, "event ID is required")
	}

	// TODO: Implement GetEvent - requires EventStore access
	// For now, return not implemented
	return nil, status.Error(codes.Unimplemented, "GetEvent not yet implemented")
}
