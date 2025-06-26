package middleware

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestCorrelationIDFromContext(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "context with correlation id",
			ctx:      context.WithValue(context.Background(), correlationIDKey{}, "test-id-123"),
			expected: "test-id-123",
		},
		{
			name:     "context without correlation id",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name: "context with metadata correlation id",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs(CorrelationIDKey, "metadata-id-456"),
			),
			expected: "metadata-id-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CorrelationIDFromContext(tt.ctx)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestContextWithCorrelationID(t *testing.T) {
	t.Run("adds correlation id to context", func(t *testing.T) {
		ctx := context.Background()
		id := "test-correlation-id"

		newCtx := ContextWithCorrelationID(ctx, id)

		// Check that the ID is in the context value
		retrieved := CorrelationIDFromContext(newCtx)
		if retrieved != id {
			t.Errorf("got %q, want %q", retrieved, id)
		}
	})

	t.Run("adds correlation id to metadata", func(t *testing.T) {
		ctx := context.Background()
		id := "test-correlation-id"

		newCtx := ContextWithCorrelationID(ctx, id)

		// Check that the ID is in the metadata
		md, ok := metadata.FromOutgoingContext(newCtx)
		if !ok {
			t.Fatal("metadata not found in context")
		}

		values := md.Get(CorrelationIDKey)
		if len(values) != 1 || values[0] != id {
			t.Errorf("got %v, want [%q]", values, id)
		}
	})
}

func TestCorrelationUnaryServerInterceptor(t *testing.T) {
	t.Run("generates correlation id if not present", func(t *testing.T) {
		interceptor := CorrelationUnaryServerInterceptor()

		var capturedCtx context.Context
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			capturedCtx = ctx
			return nil, nil
		}

		ctx := context.Background()
		interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "test"}, handler)

		// Check that a correlation ID was generated
		id := CorrelationIDFromContext(capturedCtx)
		if id == "" {
			t.Error("expected correlation id to be generated")
		}

		// Verify it's a valid UUID
		if _, err := uuid.Parse(id); err != nil {
			t.Errorf("generated correlation id is not a valid UUID: %v", err)
		}
	})

	t.Run("preserves existing correlation id", func(t *testing.T) {
		interceptor := CorrelationUnaryServerInterceptor()

		existingID := "existing-correlation-id"
		ctx := metadata.NewIncomingContext(
			context.Background(),
			metadata.Pairs(CorrelationIDKey, existingID),
		)

		var capturedCtx context.Context
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			capturedCtx = ctx
			return nil, nil
		}

		interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "test"}, handler)

		// Check that the existing correlation ID is preserved
		id := CorrelationIDFromContext(capturedCtx)
		if id != existingID {
			t.Errorf("got %q, want %q", id, existingID)
		}
	})

	t.Run("adds correlation id to outgoing metadata", func(t *testing.T) {
		interceptor := CorrelationUnaryServerInterceptor()

		var capturedCtx context.Context
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			capturedCtx = ctx
			return nil, nil
		}

		ctx := context.Background()
		interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "test"}, handler)

		// Check that the correlation ID is in the outgoing metadata
		md, ok := metadata.FromOutgoingContext(capturedCtx)
		if !ok {
			t.Fatal("outgoing metadata not found in context")
		}

		values := md.Get(CorrelationIDKey)
		if len(values) == 0 {
			t.Error("correlation id not found in outgoing metadata")
		}
	})
}

func TestCorrelationStreamServerInterceptor(t *testing.T) {
	t.Run("generates correlation id if not present", func(t *testing.T) {
		interceptor := CorrelationStreamServerInterceptor()

		mockStream := &mockServerStream{
			ctx: context.Background(),
		}

		var capturedStream grpc.ServerStream
		handler := func(srv interface{}, ss grpc.ServerStream) error {
			capturedStream = ss
			return nil
		}

		interceptor(nil, mockStream, &grpc.StreamServerInfo{FullMethod: "test"}, handler)

		// Check that a correlation ID was generated
		id := CorrelationIDFromContext(capturedStream.Context())
		if id == "" {
			t.Error("expected correlation id to be generated")
		}

		// Verify it's a valid UUID
		if _, err := uuid.Parse(id); err != nil {
			t.Errorf("generated correlation id is not a valid UUID: %v", err)
		}
	})

	t.Run("preserves existing correlation id", func(t *testing.T) {
		interceptor := CorrelationStreamServerInterceptor()

		existingID := "stream-correlation-id"
		ctx := metadata.NewIncomingContext(
			context.Background(),
			metadata.Pairs(CorrelationIDKey, existingID),
		)

		mockStream := &mockServerStream{
			ctx: ctx,
		}

		var capturedStream grpc.ServerStream
		handler := func(srv interface{}, ss grpc.ServerStream) error {
			capturedStream = ss
			return nil
		}

		interceptor(nil, mockStream, &grpc.StreamServerInfo{FullMethod: "test"}, handler)

		// Check that the existing correlation ID is preserved
		id := CorrelationIDFromContext(capturedStream.Context())
		if id != existingID {
			t.Errorf("got %q, want %q", id, existingID)
		}
	})
}

// mockServerStream is a mock implementation of grpc.ServerStream for testing
type mockServerStream struct {
	ctx context.Context
}

func (m *mockServerStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockServerStream) SendHeader(metadata.MD) error { return nil }
func (m *mockServerStream) SetTrailer(metadata.MD)       {}
func (m *mockServerStream) Context() context.Context     { return m.ctx }
func (m *mockServerStream) SendMsg(m2 interface{}) error { return nil }
func (m *mockServerStream) RecvMsg(m2 interface{}) error { return nil }
