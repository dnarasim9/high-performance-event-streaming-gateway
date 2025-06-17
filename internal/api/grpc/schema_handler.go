package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	schemav1 "github.com/dheemanth-hn/event-streaming-gateway/gen/go/schema/v1"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/dto"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/port"
)

// SchemaHandler implements the SchemaService gRPC server interface.
type SchemaHandler struct {
	schemav1.UnimplementedSchemaServiceServer
	schemaManagementUseCase port.SchemaManagementUseCase
	logger                  *slog.Logger
}

// NewSchemaHandler creates a new SchemaHandler with the specified use case.
func NewSchemaHandler(
	schemaManagementUseCase port.SchemaManagementUseCase,
	logger *slog.Logger,
) *SchemaHandler {
	return &SchemaHandler{
		schemaManagementUseCase: schemaManagementUseCase,
		logger:                  logger,
	}
}

// RegisterSchema registers a new schema in the registry.
func (h *SchemaHandler) RegisterSchema(ctx context.Context, req *schemav1.RegisterSchemaRequest) (*schemav1.RegisterSchemaResponse, error) {
	if req == nil || req.Name == "" || req.Definition == "" {
		return nil, status.Error(codes.InvalidArgument, "schema name and definition are required")
	}

	dtoReq := schemaProtoToDTO(req)
	if dtoReq == nil {
		return nil, status.Error(codes.InvalidArgument, "failed to convert request")
	}

	schemaID, err := h.schemaManagementUseCase.Register(ctx, dtoReq)
	if err != nil {
		h.logger.Error("failed to register schema", "error", err, "name", req.Name)
		return nil, status.Error(codes.Internal, "failed to register schema")
	}

	return &schemav1.RegisterSchemaResponse{
		SchemaId: schemaID,
		Success:  true,
	}, nil
}

// GetSchema retrieves a schema by ID.
func (h *SchemaHandler) GetSchema(ctx context.Context, req *schemav1.GetSchemaRequest) (*schemav1.GetSchemaResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	// Extract schema ID from the oneof identifier
	var schemaID string
	switch id := req.GetIdentifier().(type) {
	case *schemav1.GetSchemaRequest_SchemaId:
		schemaID = id.SchemaId
	default:
		return nil, status.Error(codes.InvalidArgument, "schema ID is required")
	}

	if schemaID == "" {
		return nil, status.Error(codes.InvalidArgument, "schema ID cannot be empty")
	}

	schemaDTO, err := h.schemaManagementUseCase.Get(ctx, schemaID)
	if err != nil {
		h.logger.Error("failed to get schema", "error", err, "schema_id", schemaID)
		return nil, status.Error(codes.NotFound, "schema not found")
	}

	if schemaDTO == nil {
		return nil, status.Error(codes.NotFound, "schema not found")
	}

	return &schemav1.GetSchemaResponse{
		Schema: schemaDTOToProto(schemaDTO),
	}, nil
}

// ValidateEvent validates an event against a schema.
func (h *SchemaHandler) ValidateEvent(ctx context.Context, req *schemav1.ValidateEventRequest) (*schemav1.ValidateEventResponse, error) {
	if req == nil || req.SchemaId == "" {
		return nil, status.Error(codes.InvalidArgument, "schema ID is required")
	}

	var data []byte
	if req.Event != nil {
		data = req.Event.Data
	}
	if len(data) == 0 {
		return nil, status.Error(codes.InvalidArgument, "event data is required")
	}

	dtoReq := &dto.ValidateRequest{
		SchemaID: req.SchemaId,
		Data:     data,
	}

	validationResult, err := h.schemaManagementUseCase.Validate(ctx, dtoReq)
	if err != nil {
		h.logger.Error("validation failed", "error", err, "schema_id", req.SchemaId)
		return &schemav1.ValidateEventResponse{
			Valid:        false,
			ErrorMessage: err.Error(),
		}, nil
	}

	if validationResult == nil {
		return nil, status.Error(codes.Internal, "validation returned nil result")
	}

	return &schemav1.ValidateEventResponse{
		Valid:        validationResult.Valid,
		ErrorMessage: validationResult.Error,
	}, nil
}

// ListSchemas returns schemas filtered by namespace.
func (h *SchemaHandler) ListSchemas(ctx context.Context, req *schemav1.ListSchemasRequest) (*schemav1.ListSchemasResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	schemaDTOs, err := h.schemaManagementUseCase.List(ctx, req.Namespace)
	if err != nil {
		h.logger.Error("failed to list schemas", "error", err, "namespace", req.Namespace)
		return nil, status.Error(codes.Internal, "failed to list schemas")
	}

	protoSchemas := make([]*schemav1.Schema, len(schemaDTOs))
	for i, s := range schemaDTOs {
		protoSchemas[i] = schemaDTOToProto(s)
	}

	return &schemav1.ListSchemasResponse{
		Schemas:    protoSchemas,
		TotalCount: int32(len(protoSchemas)), //nolint:gosec // safe conversion, list size will never overflow int32
	}, nil
}

// schemaDTOToProto converts a SchemaResponse DTO to a proto Schema message.
func schemaDTOToProto(s *dto.SchemaResponse) *schemav1.Schema {
	if s == nil {
		return nil
	}
	return &schemav1.Schema{
		Id:         s.ID,
		Name:       s.Name,
		Version:    int32(1),
		Format:     schemav1.SchemaFormat_SCHEMA_FORMAT_JSON_SCHEMA,
		Definition: s.Definition,
	}
}
