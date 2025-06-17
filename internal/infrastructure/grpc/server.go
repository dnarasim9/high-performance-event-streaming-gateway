package grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/auth"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/config"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/logging"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/metrics"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/ratelimiter"
)

// ServerConfig holds gRPC server configuration options.
type ServerConfig struct {
	Port                  int
	MaxConcurrentStreams  uint32
	MaxReceiveMessageSize int
	MaxSendMessageSize    int
	KeepaliveTime         time.Duration
	KeepaliveTimeout      time.Duration
	MaxConnectionIdle     time.Duration
	MaxConnectionAge      time.Duration
	MaxConnectionAgeGrace time.Duration
}

// ServiceRegistrar provides an interface for registering gRPC services.
type ServiceRegistrar interface {
	Register(*grpc.Server) error
}

// InterceptorChain represents a chain of interceptors.
type InterceptorChain struct {
	unaryInterceptors  []grpc.UnaryServerInterceptor
	streamInterceptors []grpc.StreamServerInterceptor
}

// AddUnaryInterceptor adds a unary interceptor to the chain.
func (ic *InterceptorChain) AddUnaryInterceptor(interceptor grpc.UnaryServerInterceptor) {
	ic.unaryInterceptors = append(ic.unaryInterceptors, interceptor)
}

// AddStreamInterceptor adds a stream interceptor to the chain.
func (ic *InterceptorChain) AddStreamInterceptor(interceptor grpc.StreamServerInterceptor) {
	ic.streamInterceptors = append(ic.streamInterceptors, interceptor)
}

// NewGRPCServer creates a new gRPC server with configured interceptors.
func NewGRPCServer(
	cfg *config.ServerConfig,
	logger logging.Logger,
	metricsCollector metrics.MetricsCollector,
	rateLimiter ratelimiter.RateLimiter,
	jwtAuth *auth.JWTAuthenticator,
	rbacAuth *auth.RBACAuthorizer,
) (*grpc.Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Build interceptor chain
	chain := &InterceptorChain{
		unaryInterceptors:  make([]grpc.UnaryServerInterceptor, 0),
		streamInterceptors: make([]grpc.StreamServerInterceptor, 0),
	}

	// Add logging interceptor
	chain.AddUnaryInterceptor(loggingUnaryInterceptor(logger))
	chain.AddStreamInterceptor(loggingStreamInterceptor(logger))

	// Add JWT authentication interceptor if enabled
	if jwtAuth != nil {
		chain.AddUnaryInterceptor(jwtAuth.UnaryInterceptor())
		chain.AddStreamInterceptor(jwtAuth.StreamInterceptor())
	}

	// Add RBAC authorization interceptor if enabled
	if rbacAuth != nil {
		// Note: This is a basic interceptor that logs authorization attempts
		// Actual authorization checks should be done at the handler level
		chain.AddUnaryInterceptor(rbacAuthorizationUnaryInterceptor(logger, rbacAuth))
		chain.AddStreamInterceptor(rbacAuthorizationStreamInterceptor(logger, rbacAuth))
	}

	// Add rate limiting interceptor
	if rateLimiter != nil {
		chain.AddUnaryInterceptor(rateLimitingUnaryInterceptor(logger, rateLimiter))
		chain.AddStreamInterceptor(rateLimitingStreamInterceptor(logger, rateLimiter))
	}

	// Add metrics interceptor
	if metricsCollector != nil {
		chain.AddUnaryInterceptor(metricsUnaryInterceptor(metricsCollector))
		chain.AddStreamInterceptor(metricsStreamInterceptor(metricsCollector))
	}

	// Add recovery interceptor
	chain.AddUnaryInterceptor(recoveryUnaryInterceptor(logger))
	chain.AddStreamInterceptor(recoveryStreamInterceptor(logger))

	// Create gRPC server options
	opts := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(10000),
		grpc.MaxRecvMsgSize(10 * 1024 * 1024), // 10MB
		grpc.MaxSendMsgSize(10 * 1024 * 1024), // 10MB
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    20 * time.Second,
			Timeout: 10 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.ChainUnaryInterceptor(chain.unaryInterceptors...),
		grpc.ChainStreamInterceptor(chain.streamInterceptors...),
	}

	return grpc.NewServer(opts...), nil
}

// Interceptor implementations

// loggingUnaryInterceptor logs unary RPC calls.
func loggingUnaryInterceptor(logger logging.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		startTime := time.Now()

		logger.Debug(
			"unary RPC call started",
			logging.Field{Key: "method", Value: info.FullMethod},
		)

		resp, err := handler(ctx, req)

		duration := time.Since(startTime)
		if err != nil {
			logger.Error(
				"unary RPC call failed",
				logging.Field{Key: "method", Value: info.FullMethod},
				logging.Field{Key: "duration", Value: duration.String()},
				logging.Field{Key: "error", Value: err},
			)
		} else {
			logger.Debug(
				"unary RPC call completed",
				logging.Field{Key: "method", Value: info.FullMethod},
				logging.Field{Key: "duration", Value: duration.String()},
			)
		}

		return resp, err
	}
}

// loggingStreamInterceptor logs stream RPC calls.
func loggingStreamInterceptor(logger logging.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		startTime := time.Now()

		logger.Debug(
			"stream RPC call started",
			logging.Field{Key: "method", Value: info.FullMethod},
		)

		err := handler(srv, ss)

		duration := time.Since(startTime)
		if err != nil {
			logger.Error(
				"stream RPC call failed",
				logging.Field{Key: "method", Value: info.FullMethod},
				logging.Field{Key: "duration", Value: duration.String()},
				logging.Field{Key: "error", Value: err},
			)
		} else {
			logger.Debug(
				"stream RPC call completed",
				logging.Field{Key: "method", Value: info.FullMethod},
				logging.Field{Key: "duration", Value: duration.String()},
			)
		}

		return err
	}
}

// rateLimitingUnaryInterceptor applies rate limiting to unary RPC calls.
func rateLimitingUnaryInterceptor(logger logging.Logger, limiter ratelimiter.RateLimiter) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Use client IP as key for rate limiting
		// In production, use peer info from context
		key := "default"

		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			logger.Warn("rate limiter error", logging.Field{Key: "error", Value: err})
		}

		if !allowed {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(ctx, req)
	}
}

// rateLimitingStreamInterceptor applies rate limiting to stream RPC calls.
func rateLimitingStreamInterceptor(logger logging.Logger, limiter ratelimiter.RateLimiter) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Use client IP as key for rate limiting
		key := "default"

		allowed, err := limiter.Allow(ss.Context(), key)
		if err != nil {
			logger.Warn("rate limiter error", logging.Field{Key: "error", Value: err})
		}

		if !allowed {
			return status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(srv, ss)
	}
}

// metricsUnaryInterceptor records metrics for unary RPC calls.
func metricsUnaryInterceptor(collector metrics.MetricsCollector) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		startTime := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(startTime)

		if err != nil {
			collector.RecordError("grpc_unary", "handler_error")
		}
		collector.RecordIngestionDuration(duration)

		return resp, err
	}
}

// metricsStreamInterceptor records metrics for stream RPC calls.
func metricsStreamInterceptor(collector metrics.MetricsCollector) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		startTime := time.Now()
		err := handler(srv, ss)
		duration := time.Since(startTime)

		if err != nil {
			collector.RecordError("grpc_stream", "handler_error")
		}
		collector.RecordIngestionDuration(duration)

		return err
	}
}

// rbacAuthorizationUnaryInterceptor logs RBAC authorization attempts.
func rbacAuthorizationUnaryInterceptor(logger logging.Logger, _authorizer *auth.RBACAuthorizer) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		claims, err := auth.ExtractClaims(ctx)
		if err != nil {
			logger.Debug("failed to extract claims", logging.Field{Key: "error", Value: err})
		}
		if claims != nil {
			logger.Debug(
				"RBAC check",
				logging.Field{Key: "user_id", Value: claims.UserID},
				logging.Field{Key: "method", Value: info.FullMethod},
				logging.Field{Key: "roles", Value: claims.Roles},
			)
		}

		return handler(ctx, req)
	}
}

// rbacAuthorizationStreamInterceptor logs RBAC authorization attempts for streams.
func rbacAuthorizationStreamInterceptor(logger logging.Logger, _authorizer *auth.RBACAuthorizer) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		claims, err := auth.ExtractClaims(ss.Context())
		if err != nil {
			logger.Debug("failed to extract claims", logging.Field{Key: "error", Value: err})
		}
		if claims != nil {
			logger.Debug(
				"RBAC check",
				logging.Field{Key: "user_id", Value: claims.UserID},
				logging.Field{Key: "method", Value: info.FullMethod},
				logging.Field{Key: "roles", Value: claims.Roles},
			)
		}

		return handler(srv, ss)
	}
}

// recoveryUnaryInterceptor recovers from panics in unary RPC handlers.
func recoveryUnaryInterceptor(logger logging.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error(
					"panic in RPC handler",
					logging.Field{Key: "method", Value: info.FullMethod},
					logging.Field{Key: "panic", Value: r},
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// recoveryStreamInterceptor recovers from panics in stream RPC handlers.
func recoveryStreamInterceptor(logger logging.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error(
					"panic in stream RPC handler",
					logging.Field{Key: "method", Value: info.FullMethod},
					logging.Field{Key: "panic", Value: r},
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(srv, ss)
	}
}
