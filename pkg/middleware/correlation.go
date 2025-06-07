package middleware

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// CorrelationIDKey is the context key for storing correlation IDs
const CorrelationIDKey = "x-correlation-id"

type correlationIDKey struct{}

// CorrelationIDFromContext extracts the correlation ID from the context.
// If no correlation ID is found, it returns an empty string.
func CorrelationIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey{}).(string); ok {
		return id
	}

	// Try to get from gRPC metadata as fallback
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(CorrelationIDKey); len(values) > 0 {
			return values[0]
		}
	}

	return ""
}

// ContextWithCorrelationID adds a correlation ID to the context.
// It stores the ID in the context value and also in the outgoing metadata.
func ContextWithCorrelationID(ctx context.Context, id string) context.Context {
	// Store in context value
	ctx = context.WithValue(ctx, correlationIDKey{}, id)

	// Add to outgoing metadata
	md, _ := metadata.FromOutgoingContext(ctx)
	if md == nil {
		md = metadata.MD{}
	}
	md.Set(CorrelationIDKey, id)

	return metadata.NewOutgoingContext(ctx, md)
}

// CorrelationUnaryServerInterceptor is a gRPC unary server interceptor that
// extracts or generates a correlation ID from request metadata and adds it to the context.
// This enables request tracing across the system.
func CorrelationUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		correlationID := extractOrGenerateCorrelationID(ctx)
		ctx = context.WithValue(ctx, correlationIDKey{}, correlationID)

		// Add correlation ID to outgoing metadata for downstream services
		md, _ := metadata.FromIncomingContext(ctx)
		if md == nil {
			md = metadata.MD{}
		}
		md.Set(CorrelationIDKey, correlationID)
		ctx = metadata.NewOutgoingContext(ctx, md)

		return handler(ctx, req)
	}
}

// CorrelationStreamServerInterceptor is a gRPC stream server interceptor that
// extracts or generates a correlation ID from request metadata and adds it to the context.
// This enables request tracing for streaming RPCs.
func CorrelationStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := ss.Context()
		correlationID := extractOrGenerateCorrelationID(ctx)
		ctx = context.WithValue(ctx, correlationIDKey{}, correlationID)

		// Add correlation ID to outgoing metadata for downstream services
		md, _ := metadata.FromIncomingContext(ctx)
		if md == nil {
			md = metadata.MD{}
		}
		md.Set(CorrelationIDKey, correlationID)
		ctx = metadata.NewOutgoingContext(ctx, md)

		// Create wrapped stream with new context
		wrappedStream := &serverStreamWrapper{
			ServerStream: ss,
			ctx:          ctx,
		}

		return handler(srv, wrappedStream)
	}
}

// extractOrGenerateCorrelationID extracts the correlation ID from incoming metadata,
// or generates a new UUID if none is present.
func extractOrGenerateCorrelationID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return uuid.New().String()
	}

	if values := md.Get(CorrelationIDKey); len(values) > 0 && values[0] != "" {
		return values[0]
	}

	return uuid.New().String()
}

// serverStreamWrapper wraps a gRPC ServerStream and overrides its Context method
// to return the modified context with the correlation ID.
type serverStreamWrapper struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the wrapped context.
func (w *serverStreamWrapper) Context() context.Context {
	return w.ctx
}
