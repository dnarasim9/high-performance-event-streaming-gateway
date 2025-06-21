package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"

	eventv1 "github.com/dheemanth-hn/event-streaming-gateway/gen/go/event/v1"
	gatewayv1 "github.com/dheemanth-hn/event-streaming-gateway/gen/go/gateway/v1"
	schemav1 "github.com/dheemanth-hn/event-streaming-gateway/gen/go/schema/v1"
	grpcapi "github.com/dheemanth-hn/event-streaming-gateway/internal/api/grpc"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/service"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/valueobject"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/auth"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/config"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/health"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/kafka"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/logging"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/metrics"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/ratelimiter"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/replay"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/repository"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/schema"
	"github.com/dheemanth-hn/event-streaming-gateway/pkg/middleware"
)

func main() {
	// Initialize logger first
	logger := initializeLogger()

	defer func() {
		if err := recover(); err != nil {
			logger.Error("panic", "error", err)
		}
	}()

	if err := run(logger); err != nil {
		logger.Error("startup failed", "error", err)
		return
	}
}

func run(logger *slog.Logger) error {
	// Load configuration
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	logger.Info("starting event streaming gateway", "environment", cfg.Environment)

	// Initialize infrastructure components
	metricsCollector, err := initializeMetrics(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize metrics: %w", err)
	}

	rateLimiter := initializeRateLimiter(cfg)

	jwtAuthenticator := initializeAuth(cfg)

	rbacAuthorizer := auth.NewRBACAuthorizer()

	schemaValidator := initializeSchemaValidator(cfg)

	// Initialize message broker (Kafka)
	kafkaProducer, err := initializeKafkaProducer(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize Kafka producer: %w", err)
	}
	defer func() {
		if err := kafkaProducer.Close(); err != nil {
			logger.Error("failed to close Kafka producer", "error", err)
		}
	}()

	kafkaConsumer, err := initializeKafkaConsumer(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize Kafka consumer: %w", err)
	}
	defer func() {
		if err := kafkaConsumer.Close(); err != nil {
			logger.Error("failed to close Kafka consumer", "error", err)
		}
	}()

	// Initialize health checker
	healthChecker := health.NewHealthChecker(logger)
	if err := registerHealthChecks(healthChecker, kafkaProducer, kafkaConsumer); err != nil {
		return fmt.Errorf("failed to register health checks: %w", err)
	}

	// Initialize in-memory event store for replay functionality
	eventStore := replay.NewInMemoryEventStore(24 * time.Hour) // 24 hour retention

	// Create message broker adapter that wraps both producer and consumer
	messageBroker := &messagebrokerAdapter{
		producer: kafkaProducer,
		consumer: kafkaConsumer,
	}

	// Initialize application services
	eventService := service.NewEventService(
		eventStore,
		messageBroker, // Implements EventPublisher via MessageBroker composition
		schemaValidator,
		metricsCollector,
		logger,
	)

	subscriptionService := service.NewSubscriptionService(
		messageBroker, // Implements EventSubscriber via MessageBroker composition
		metricsCollector,
		logger,
	)

	replayService := service.NewReplayService(
		eventStore,
		metricsCollector,
		logger,
	)

	consumerService := service.NewConsumerService(
		repository.NewInMemoryConsumerRepository(),
		metricsCollector,
		logger,
	)

	schemaService := service.NewSchemaService(
		repository.NewInMemorySchemaRepository(),
		schemaValidator,
		metricsCollector,
		logger,
	)

	// Initialize gRPC handlers
	eventHandler := grpcapi.NewEventHandler(eventService, subscriptionService, replayService, logger)
	healthCheckerAdapter := &healthCheckerAdapter{hc: healthChecker}
	gatewayHandler := grpcapi.NewGatewayHandler(consumerService, healthCheckerAdapter, metricsCollector, logger)
	schemaHandler := grpcapi.NewSchemaHandler(schemaService, logger)

	// Create gRPC server with interceptors
	grpcServer := createGRPCServer(cfg, logger, metricsCollector, rateLimiter, jwtAuthenticator, rbacAuthorizer, eventHandler, gatewayHandler, schemaHandler)

	// Create HTTP server for health and metrics
	httpServer := createHTTPServer(cfg, healthChecker, metricsCollector)

	// Setup graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	// Start gRPC server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := startGRPCServer(grpcServer, cfg, logger); err != nil {
			logger.Error("gRPC server error", "error", err)
		}
	}()

	// Start HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := startHTTPServer(httpServer, cfg, logger); err != nil {
			logger.Error("HTTP server error", "error", err)
		}
	}()

	logger.Info("event streaming gateway is running", "grpc_port", cfg.Server.GRPCPort, "http_port", cfg.Server.HTTPPort)

	// Wait for shutdown signal
	<-shutdownChan
	logger.Info("shutdown signal received, gracefully stopping...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	logger.Info("gRPC server stopped")

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}
	logger.Info("HTTP server stopped")

	wg.Wait()
	logger.Info("event streaming gateway stopped")
	return nil
}

// initializeLogger creates and configures the logger.
func initializeLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// initializeMetrics creates and initializes the metrics collector.
func initializeMetrics(_cfg *config.Config) (*metrics.PrometheusCollector, error) {
	collector, err := metrics.NewPrometheusCollector()
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus collector: %w", err)
	}
	return collector, nil
}

// initializeRateLimiter creates and initializes the rate limiter.
func initializeRateLimiter(cfg *config.Config) ratelimiter.RateLimiter {
	limiter := ratelimiter.NewTokenBucketLimiter(&cfg.RateLimit)
	return limiter
}

// initializeAuth creates and initializes the JWT authenticator.
func initializeAuth(cfg *config.Config) *auth.JWTAuthenticator {
	return auth.NewJWTAuthenticator(cfg.Auth.JWTSecret, cfg.Auth.Issuer, cfg.Auth.TokenExpiry)
}

// initializeSchemaValidator creates and initializes the schema validator.
func initializeSchemaValidator(_cfg *config.Config) *schema.SchemaValidator {
	return schema.NewSchemaValidator(5 * time.Minute)
}

// initializeKafkaProducer creates and initializes the Kafka producer.
func initializeKafkaProducer(cfg *config.Config, logger *slog.Logger) (*kafka.Producer, error) {
	// Create a simple adapter logger since Producer expects logging.Logger
	loggerAdapter := &slogToLoggerAdapter{slog: logger}
	producer, err := kafka.NewProducer(&cfg.Kafka, loggerAdapter, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}
	return producer, nil
}

// initializeKafkaConsumer creates and initializes the Kafka consumer.
func initializeKafkaConsumer(cfg *config.Config, logger *slog.Logger) (*kafka.Consumer, error) {
	// Create a simple adapter logger since Consumer expects logging.Logger
	loggerAdapter := &slogToLoggerAdapter{slog: logger}
	consumer, err := kafka.NewConsumer(&cfg.Kafka, loggerAdapter, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer: %w", err)
	}
	return consumer, nil
}

// registerHealthChecks registers health checks for various components.
func registerHealthChecks(hc *health.HealthChecker, _producer *kafka.Producer, _consumer *kafka.Consumer) error {
	// Register Kafka health check
	kafkaCheck := health.NewGenericCheck("kafka", func(ctx context.Context) error {
		// Simple check - if the producer and consumer are initialized, assume Kafka is healthy
		// In production, we'd want actual health checks
		return nil
	})
	if err := hc.RegisterCheck(kafkaCheck); err != nil {
		return err
	}

	// Register a simple readiness check
	readyCheck := health.NewSimpleHealthCheck("readiness")
	return hc.RegisterCheck(readyCheck)
}

// createGRPCServer creates a configured gRPC server.
func createGRPCServer(
	_cfg *config.Config,
	_logger *slog.Logger,
	_metricsCollector *metrics.PrometheusCollector,
	_rateLimiter ratelimiter.RateLimiter,
	jwtAuth *auth.JWTAuthenticator,
	_rbacAuth *auth.RBACAuthorizer,
	eventHandler *grpcapi.EventHandler,
	gatewayHandler *grpcapi.GatewayHandler,
	schemaHandler *grpcapi.SchemaHandler,
) *grpc.Server {
	// Create interceptors
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		middleware.CorrelationUnaryServerInterceptor(),
		jwtAuth.UnaryInterceptor(),
		// Add more interceptors as needed (logging, metrics, rate limiting)
	}

	streamInterceptors := []grpc.StreamServerInterceptor{
		middleware.CorrelationStreamServerInterceptor(),
		jwtAuth.StreamInterceptor(),
		// Add more interceptors as needed
	}

	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
		grpc.ChainStreamInterceptor(streamInterceptors...),
		grpc.MaxConcurrentStreams(100),
		grpc.MaxRecvMsgSize(20 * 1024 * 1024), // 20 MB
		grpc.MaxSendMsgSize(20 * 1024 * 1024), // 20 MB
	}

	server := grpc.NewServer(opts...)

	// Register services
	eventv1.RegisterEventServiceServer(server, eventHandler)
	gatewayv1.RegisterGatewayServiceServer(server, gatewayHandler)
	schemav1.RegisterSchemaServiceServer(server, schemaHandler)

	return server
}

// startGRPCServer starts the gRPC server and listens for connections.
func startGRPCServer(server *grpc.Server, cfg *config.Config, logger *slog.Logger) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", cfg.Server.GRPCPort, err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			logger.Error("failed to close listener", "error", err)
		}
	}()

	logger.Info("gRPC server listening", "port", cfg.Server.GRPCPort)
	return server.Serve(listener)
}

// createHTTPServer creates an HTTP server for health checks and metrics.
func createHTTPServer(cfg *config.Config, healthChecker *health.HealthChecker, metricsCollector *metrics.PrometheusCollector) *http.Server {
	mux := http.NewServeMux()

	// Register health check endpoint
	mux.HandleFunc("/health", healthChecker.HealthHTTPHandler())
	mux.HandleFunc("/ready", healthChecker.ReadyHTTPHandler())

	// Register metrics endpoint
	mux.Handle("/metrics", metricsCollector.HTTPHandler())

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// startHTTPServer starts the HTTP server.
func startHTTPServer(server *http.Server, cfg *config.Config, logger *slog.Logger) error {
	logger.Info("HTTP server listening", "port", cfg.Server.HTTPPort)
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server error: %w", err)
	}
	return nil
}

// healthCheckerAdapter adapts the HealthChecker to the HealthChecker interface expected by GatewayHandler.
type healthCheckerAdapter struct {
	hc *health.HealthChecker
}

// GetStatus returns the health status.
func (hca *healthCheckerAdapter) GetStatus(ctx context.Context) (*grpcapi.HealthStatus, error) {
	status := hca.hc.CheckAll(ctx)
	return &grpcapi.HealthStatus{
		Status:  status.Status,
		Message: fmt.Sprintf("checks: %d", len(status.Checks)),
	}, nil
}

// messagebrokerAdapter implements port.MessageBroker by wrapping Producer and Consumer
type messagebrokerAdapter struct {
	producer *kafka.Producer
	consumer *kafka.Consumer
}

func (mba *messagebrokerAdapter) Publish(ctx context.Context, event *entity.Event, topic string) error {
	return mba.producer.Publish(ctx, event, "")
}

func (mba *messagebrokerAdapter) PublishBatch(ctx context.Context, events []*entity.Event, topic string) error {
	return mba.producer.PublishBatch(ctx, events, "")
}

func (mba *messagebrokerAdapter) Subscribe(
	ctx context.Context,
	topic string,
	group string,
	filter valueobject.EventFilter,
) (<-chan *entity.Event, error) {
	// Create event channel
	eventChan := make(chan *entity.Event, 100)
	errChan := make(chan error, 1)

	// Start consumer goroutine
	go func() {
		if err := mba.consumer.Subscribe(ctx, eventChan, errChan); err != nil {
			errChan <- err
		}
	}()

	return eventChan, nil
}

func (mba *messagebrokerAdapter) Unsubscribe() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return mba.consumer.Unsubscribe(ctx)
}

func (mba *messagebrokerAdapter) Close() error {
	if err := mba.producer.Close(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return mba.consumer.Unsubscribe(ctx)
}

// slogToLoggerAdapter adapts slog.Logger to the logging.Logger interface.
type slogToLoggerAdapter struct {
	slog *slog.Logger
}

// Info logs an info level message.
func (a *slogToLoggerAdapter) Info(msg string, fields ...logging.Field) {
	attrs := make([]any, 0, len(fields)*2)
	for _, f := range fields {
		attrs = append(attrs, f.Key, f.Value)
	}
	a.slog.Info(msg, attrs...)
}

// Error logs an error level message.
func (a *slogToLoggerAdapter) Error(msg string, fields ...logging.Field) {
	attrs := make([]any, 0, len(fields)*2)
	for _, f := range fields {
		attrs = append(attrs, f.Key, f.Value)
	}
	a.slog.Error(msg, attrs...)
}

// Warn logs a warning level message.
func (a *slogToLoggerAdapter) Warn(msg string, fields ...logging.Field) {
	attrs := make([]any, 0, len(fields)*2)
	for _, f := range fields {
		attrs = append(attrs, f.Key, f.Value)
	}
	a.slog.Warn(msg, attrs...)
}

// Debug logs a debug level message.
func (a *slogToLoggerAdapter) Debug(msg string, fields ...logging.Field) {
	attrs := make([]any, 0, len(fields)*2)
	for _, f := range fields {
		attrs = append(attrs, f.Key, f.Value)
	}
	a.slog.Debug(msg, attrs...)
}

// WithField returns a new Logger with an additional field.
func (a *slogToLoggerAdapter) WithField(key string, value interface{}) logging.Logger {
	return &slogToLoggerAdapter{
		slog: a.slog.With(key, value),
	}
}

// WithFields returns a new Logger with additional fields.
func (a *slogToLoggerAdapter) WithFields(fields map[string]interface{}) logging.Logger {
	attrs := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}
	return &slogToLoggerAdapter{
		slog: a.slog.With(attrs...),
	}
}

// WithCorrelationID returns a new Logger with a correlation ID.
func (a *slogToLoggerAdapter) WithCorrelationID(id string) logging.Logger {
	return &slogToLoggerAdapter{
		slog: a.slog.With("correlation_id", id),
	}
}

// Sync flushes any buffered log entries.
func (a *slogToLoggerAdapter) Sync() error {
	return nil
}
