package grpc

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	eventv1 "github.com/dheemanth-hn/event-streaming-gateway/gen/go/event/v1"
	gatewayv1 "github.com/dheemanth-hn/event-streaming-gateway/gen/go/gateway/v1"
	schemav1 "github.com/dheemanth-hn/event-streaming-gateway/gen/go/schema/v1"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/dto"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
)

// eventProtoToDTO converts a proto IngestEventRequest message to a DTO.
func eventProtoToDTO(protoReq *eventv1.IngestEventRequest) *dto.IngestEventRequest {
	if protoReq == nil {
		return nil
	}

	// Extract the Event sub-message
	protoEvent := protoReq.Event
	if protoEvent == nil {
		return nil
	}

	metadata := make(map[string]string)
	if protoEvent.Metadata != nil {
		metadata = protoEvent.Metadata
	}

	return &dto.IngestEventRequest{
		Source:          protoEvent.Source,
		Type:            protoEvent.Type,
		Subject:         protoEvent.Subject,
		Data:            protoEvent.Data,
		DataContentType: protoEvent.DataContentType,
		SchemaURL:       protoEvent.SchemaUrl,
		Metadata:        metadata,
		CorrelationID:   protoEvent.CorrelationId,
		CausationID:     protoEvent.CausationId,
		PartitionKey:    protoEvent.PartitionKey,
		Priority:        int(protoEvent.Priority),
	}
}

// eventEntityToProto converts a domain Event entity to a proto Event message.
func eventEntityToProto(entity *entity.Event) *eventv1.Event {
	if entity == nil {
		return nil
	}

	metadata := make(map[string]string)
	if entity.Metadata() != nil {
		metadata = entity.Metadata()
	}

	return &eventv1.Event{
		Id:              entity.ID(),
		Source:          entity.Source(),
		Type:            entity.Type(),
		Subject:         entity.Subject(),
		Data:            entity.Data(),
		DataContentType: entity.DataContentType(),
		SchemaUrl:       entity.SchemaURL(),
		Timestamp:       timestamppb.New(entity.Timestamp()),
		Metadata:        metadata,
		CorrelationId:   entity.CorrelationID(),
		CausationId:     entity.CausationID(),
		PartitionKey:    entity.PartitionKey(),
		Priority:        priorityDomainToProto(entity.Priority()),
	}
}

// subscribeRequestProtoToDTO converts a proto SubscribeRequest to a DTO.
func subscribeRequestProtoToDTO(protoReq *eventv1.SubscribeRequest) *dto.SubscribeRequest {
	if protoReq == nil {
		return nil
	}

	filterDTO := &dto.EventFilterDTO{
		SourcePattern:   "",
		TypePattern:     "",
		SubjectPattern:  "",
		MetadataFilters: make(map[string]string),
		MinPriority:     0,
	}

	if protoReq.Filter != nil {
		filterDTO.SourcePattern = protoReq.Filter.SourcePattern
		filterDTO.TypePattern = protoReq.Filter.TypePattern
		filterDTO.SubjectPattern = protoReq.Filter.SubjectPattern
		if protoReq.Filter.MetadataFilters != nil {
			filterDTO.MetadataFilters = protoReq.Filter.MetadataFilters
		}
		filterDTO.MinPriority = int(protoReq.Filter.PriorityFilter)
	}

	return &dto.SubscribeRequest{
		Filter: filterDTO,
	}
}

// replayRequestProtoToDTO converts a proto ReplayRequest to a DTO.
func replayRequestProtoToDTO(protoReq *eventv1.ReplayRequest) *dto.ReplayRequest {
	if protoReq == nil {
		return nil
	}

	filterDTO := &dto.EventFilterDTO{
		SourcePattern:   "",
		TypePattern:     "",
		SubjectPattern:  "",
		MetadataFilters: make(map[string]string),
		MinPriority:     0,
	}

	if protoReq.Filter != nil {
		filterDTO.SourcePattern = protoReq.Filter.SourcePattern
		filterDTO.TypePattern = protoReq.Filter.TypePattern
		filterDTO.SubjectPattern = protoReq.Filter.SubjectPattern
		if protoReq.Filter.MetadataFilters != nil {
			filterDTO.MetadataFilters = protoReq.Filter.MetadataFilters
		}
		filterDTO.MinPriority = int(protoReq.Filter.PriorityFilter)
	}

	return &dto.ReplayRequest{
		StartTime: protoReq.StartTime.AsTime(),
		EndTime:   protoReq.EndTime.AsTime(),
		Filter:    filterDTO,
	}
}

// priorityDomainToProto converts a domain Priority to a proto Priority.
func priorityDomainToProto(priority entity.Priority) eventv1.Priority {
	switch priority {
	case entity.PriorityLow:
		return eventv1.Priority_PRIORITY_LOW
	case entity.PriorityMedium:
		return eventv1.Priority_PRIORITY_MEDIUM
	case entity.PriorityHigh:
		return eventv1.Priority_PRIORITY_HIGH
	case entity.PriorityCritical:
		return eventv1.Priority_PRIORITY_CRITICAL
	default:
		return eventv1.Priority_PRIORITY_UNSPECIFIED
	}
}

// consumerProtoToDTO converts a proto RegisterConsumerRequest to a DTO.
func consumerProtoToDTO(protoReq *gatewayv1.RegisterConsumerRequest) *dto.RegisterConsumerRequest {
	if protoReq == nil {
		return nil
	}

	// Extract the nested Consumer message
	consumer := protoReq.Consumer
	if consumer == nil {
		return &dto.RegisterConsumerRequest{
			Name:            "",
			GroupID:         "",
			FilterConfig:    &dto.EventFilterDTO{},
			TransformConfig: &dto.TransformationConfigDTO{},
		}
	}

	filterDTO := &dto.EventFilterDTO{
		SourcePattern:   "",
		TypePattern:     "",
		SubjectPattern:  "",
		MetadataFilters: make(map[string]string),
		MinPriority:     0,
	}

	transformConfig := &dto.TransformationConfigDTO{}

	// Convert routing rules if present
	if len(consumer.RoutingRules) > 0 && consumer.RoutingRules[0] != nil {
		rule := consumer.RoutingRules[0]
		if rule.Filter != nil {
			filterDTO.SourcePattern = rule.Filter.SourcePattern
			filterDTO.TypePattern = rule.Filter.TypePattern
			filterDTO.SubjectPattern = rule.Filter.SubjectPattern
			if rule.Filter.MetadataFilters != nil {
				filterDTO.MetadataFilters = rule.Filter.MetadataFilters
			}
			filterDTO.MinPriority = int(rule.Filter.PriorityFilter)
		}
		if rule.Transformation != nil {
			transformConfig.Type = rule.Transformation.Type.String()
			transformConfig.Expression = rule.Transformation.Expression
		}
	}

	return &dto.RegisterConsumerRequest{
		Name:            consumer.Name,
		GroupID:         "",
		FilterConfig:    filterDTO,
		TransformConfig: transformConfig,
	}
}

// schemaProtoToDTO converts a proto RegisterSchemaRequest to a DTO.
func schemaProtoToDTO(protoReq *schemav1.RegisterSchemaRequest) *dto.RegisterSchemaRequest {
	if protoReq == nil {
		return nil
	}

	return &dto.RegisterSchemaRequest{
		Name:       protoReq.Name,
		Version:    "1.0", // Default version for new schemas
		Format:     protoReq.Format.String(),
		Definition: protoReq.Definition,
	}
}
