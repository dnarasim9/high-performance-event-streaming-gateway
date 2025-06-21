package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ServerConfig holds gRPC and HTTP server configuration.
type ServerConfig struct {
	GRPCPort int
	HTTPPort int
}

// KafkaConfig holds Kafka connection and topic configuration.
type KafkaConfig struct {
	Brokers           []string
	Topic             string
	GroupID           string
	NumPartitions     int32
	ReplicationFactor int16
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	DialTimeout       time.Duration
	CommitInterval    time.Duration
	SessionTimeout    time.Duration
}

// AuthConfig holds JWT and authentication configuration.
type AuthConfig struct {
	Enabled       bool
	JWTSecret     string
	Issuer        string
	TokenExpiry   time.Duration
	AllowedIssuer string
}

// RateLimitConfig holds rate limiting configuration.
type RateLimitConfig struct {
	Enabled           bool
	RequestsPerSecond int
	BurstSize         int
	CleanupInterval   time.Duration
	StaleThreshold    time.Duration
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level  string // "debug", "info", "warn", "error"
	Format string // "json", "text"
	Output string // "stdout", "stderr", or file path
}

// MetricsConfig holds Prometheus metrics configuration.
type MetricsConfig struct {
	Enabled bool
	Port    int
	Path    string
}

// ReplayConfig holds event replay configuration.
type ReplayConfig struct {
	MaxDuration  time.Duration
	BatchSize    int
	RateLimit    int // events per second
	MaxRetention time.Duration
}

// SchemaConfig holds schema validation configuration.
type SchemaConfig struct {
	ValidationEnabled bool
	RegistryURL       string
	CacheTTL          time.Duration
}

// Config holds all application configuration.
type Config struct {
	Server      ServerConfig
	Kafka       KafkaConfig
	Auth        AuthConfig
	RateLimit   RateLimitConfig
	Log         LogConfig
	Metrics     MetricsConfig
	Replay      ReplayConfig
	Schema      SchemaConfig
	Environment string
}

// Load reads configuration from environment variables and returns a Config.
func Load() *Config {
	cfg := &Config{
		Server: ServerConfig{
			GRPCPort: getIntEnv("GRPC_PORT", 50051),
			HTTPPort: getIntEnv("HTTP_PORT", 8080),
		},
		Kafka: KafkaConfig{
			Brokers:           getStringSliceEnv("KAFKA_BROKERS", []string{"localhost:9092"}),
			Topic:             getStringEnv("KAFKA_TOPIC", "events"),
			GroupID:           getStringEnv("KAFKA_GROUP_ID", "event-gateway"),
			NumPartitions:     int32(getIntEnv("KAFKA_NUM_PARTITIONS", 3)),     //nolint:gosec // safe conversion, values will never overflow int32
			ReplicationFactor: int16(getIntEnv("KAFKA_REPLICATION_FACTOR", 1)), //nolint:gosec // safe conversion, values will never overflow int16
			ReadTimeout:       getDurationEnv("KAFKA_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:      getDurationEnv("KAFKA_WRITE_TIMEOUT", 10*time.Second),
			DialTimeout:       getDurationEnv("KAFKA_DIAL_TIMEOUT", 10*time.Second),
			CommitInterval:    getDurationEnv("KAFKA_COMMIT_INTERVAL", 1*time.Second),
			SessionTimeout:    getDurationEnv("KAFKA_SESSION_TIMEOUT", 10*time.Second),
		},
		Auth: AuthConfig{
			Enabled:       getBoolEnv("AUTH_ENABLED", false),
			JWTSecret:     getStringEnv("JWT_SECRET", "default-secret-change-in-production"),
			Issuer:        getStringEnv("JWT_ISSUER", "event-gateway"),
			TokenExpiry:   getDurationEnv("JWT_TOKEN_EXPIRY", 24*time.Hour),
			AllowedIssuer: getStringEnv("JWT_ALLOWED_ISSUER", "event-gateway"),
		},
		RateLimit: RateLimitConfig{
			Enabled:           getBoolEnv("RATE_LIMIT_ENABLED", true),
			RequestsPerSecond: getIntEnv("RATE_LIMIT_RPS", 1000),
			BurstSize:         getIntEnv("RATE_LIMIT_BURST", 100),
			CleanupInterval:   getDurationEnv("RATE_LIMIT_CLEANUP_INTERVAL", 5*time.Minute),
			StaleThreshold:    getDurationEnv("RATE_LIMIT_STALE_THRESHOLD", 10*time.Minute),
		},
		Log: LogConfig{
			Level:  getStringEnv("LOG_LEVEL", "info"),
			Format: getStringEnv("LOG_FORMAT", "json"),
			Output: getStringEnv("LOG_OUTPUT", "stdout"),
		},
		Metrics: MetricsConfig{
			Enabled: getBoolEnv("METRICS_ENABLED", true),
			Port:    getIntEnv("METRICS_PORT", 9090),
			Path:    getStringEnv("METRICS_PATH", "/metrics"),
		},
		Replay: ReplayConfig{
			MaxDuration:  getDurationEnv("REPLAY_MAX_DURATION", 7*24*time.Hour),
			BatchSize:    getIntEnv("REPLAY_BATCH_SIZE", 1000),
			RateLimit:    getIntEnv("REPLAY_RATE_LIMIT", 1000),
			MaxRetention: getDurationEnv("REPLAY_MAX_RETENTION", 7*24*time.Hour),
		},
		Schema: SchemaConfig{
			ValidationEnabled: getBoolEnv("SCHEMA_VALIDATION_ENABLED", false),
			RegistryURL:       getStringEnv("SCHEMA_REGISTRY_URL", "http://localhost:8081"),
			CacheTTL:          getDurationEnv("SCHEMA_CACHE_TTL", 1*time.Hour),
		},
		Environment: getStringEnv("ENVIRONMENT", "development"),
	}

	return cfg
}

// Validate checks the configuration for validity.
func (c *Config) Validate() error {
	if err := c.validateServer(); err != nil {
		return err
	}
	if err := c.validateKafka(); err != nil {
		return err
	}
	if err := c.validateAuth(); err != nil {
		return err
	}
	if err := c.validateRateLimit(); err != nil {
		return err
	}
	return c.validateLogging()
}

// validateServer validates server configuration.
func (c *Config) validateServer() error {
	if c.Server.GRPCPort <= 0 || c.Server.GRPCPort > 65535 {
		return fmt.Errorf("invalid GRPC_PORT: %d", c.Server.GRPCPort)
	}

	if c.Server.HTTPPort <= 0 || c.Server.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP_PORT: %d", c.Server.HTTPPort)
	}
	return nil
}

// validateKafka validates Kafka configuration.
func (c *Config) validateKafka() error {
	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("at least one Kafka broker must be configured")
	}

	if c.Kafka.Topic == "" {
		return fmt.Errorf("Kafka topic must be configured")
	}

	if c.Kafka.GroupID == "" {
		return fmt.Errorf("Kafka group ID must be configured")
	}

	if c.Kafka.NumPartitions < 1 {
		return fmt.Errorf("Kafka partitions must be at least 1")
	}
	return nil
}

// validateAuth validates auth configuration.
func (c *Config) validateAuth() error {
	if c.Auth.Enabled && c.Auth.JWTSecret == "default-secret-change-in-production" {
		return fmt.Errorf("JWT secret must be changed in production (JWT_SECRET env var)")
	}
	return nil
}

// validateRateLimit validates rate limit configuration.
func (c *Config) validateRateLimit() error {
	if c.RateLimit.RequestsPerSecond < 1 {
		return fmt.Errorf("rate limit requests per second must be at least 1")
	}
	return nil
}

// validateLogging validates logging configuration.
func (c *Config) validateLogging() error {
	if c.Log.Level != "debug" && c.Log.Level != "info" && c.Log.Level != "warn" && c.Log.Level != "error" {
		return fmt.Errorf("invalid log level: %s", c.Log.Level)
	}
	return nil
}

// Helper functions for environment variable parsing

func getStringEnv(key, defaultVal string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return defaultVal
}

func getIntEnv(key string, defaultVal int) int {
	if val, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getBoolEnv(key string, defaultVal bool) bool {
	if val, exists := os.LookupEnv(key); exists {
		return val == "true" || val == "1" || val == "yes"
	}
	return defaultVal
}

func getDurationEnv(key string, defaultVal time.Duration) time.Duration {
	if val, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(val); err == nil {
			return duration
		}
	}
	return defaultVal
}

func getStringSliceEnv(key string, defaultVal []string) []string {
	if val, exists := os.LookupEnv(key); exists {
		parts := strings.Split(val, ",")
		trimmed := make([]string, len(parts))
		for i, p := range parts {
			trimmed[i] = strings.TrimSpace(p)
		}
		return trimmed
	}
	return defaultVal
}
