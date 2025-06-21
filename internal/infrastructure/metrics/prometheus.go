package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsCollector defines the interface for collecting metrics.
type MetricsCollector interface { //nolint:revive // established API
	RecordEventIngested()
	RecordEventPublished()
	RecordEventSubscribed()
	RecordError(component, errorType string)
	RecordIngestionDuration(duration time.Duration)
	RecordPublishDuration(duration time.Duration)
	RecordActiveSubscription(delta int)
	RecordActiveConsumer(delta int)
	HTTPHandler() http.Handler
}

// PrometheusCollector implements MetricsCollector using Prometheus.
type PrometheusCollector struct {
	eventsIngestedTotal      prometheus.Counter
	eventsPublishedTotal     prometheus.Counter
	eventsSubscribedTotal    prometheus.Counter
	errorsTotal              prometheus.CounterVec
	ingestionDurationSeconds prometheus.Histogram
	publishDurationSeconds   prometheus.Histogram
	activeSubscriptions      prometheus.Gauge
	activeConsumers          prometheus.Gauge
	latencyHistogram         prometheus.HistogramVec
	subscriptionGaugeVec     prometheus.GaugeVec
}

// NewPrometheusCollector creates and registers a new PrometheusCollector.
func NewPrometheusCollector() (*PrometheusCollector, error) {
	pc := &PrometheusCollector{
		eventsIngestedTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "events_ingested_total",
				Help: "Total number of events ingested",
			},
		),
		eventsPublishedTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "events_published_total",
				Help: "Total number of events published to Kafka",
			},
		),
		eventsSubscribedTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "events_subscribed_total",
				Help: "Total number of events subscribed from Kafka",
			},
		),
		errorsTotal: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "errors_total",
				Help: "Total number of errors by component and type",
			},
			[]string{"component", "error_type"},
		),
		ingestionDurationSeconds: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "event_ingestion_duration_seconds",
				Help:    "Duration of event ingestion in seconds",
				Buckets: []float64{0.001, 0.01, 0.05, 0.1, 0.5, 1.0, 2.5, 5.0, 10.0},
			},
		),
		publishDurationSeconds: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "event_publish_duration_seconds",
				Help:    "Duration of event publishing in seconds",
				Buckets: []float64{0.001, 0.01, 0.05, 0.1, 0.5, 1.0, 2.5, 5.0, 10.0},
			},
		),
		activeSubscriptions: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "active_subscriptions",
				Help: "Number of active subscriptions",
			},
		),
		activeConsumers: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "active_consumers",
				Help: "Number of active Kafka consumers",
			},
		),
		latencyHistogram: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "operation_latency_milliseconds",
				Help:    "Latency of operations in milliseconds",
				Buckets: []float64{1, 5, 10, 50, 100, 500, 1000, 5000},
			},
			[]string{"operation"},
		),
		subscriptionGaugeVec: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "subscriptions_active",
				Help: "Number of active subscriptions by consumer and group",
			},
			[]string{"consumer_id", "group_id"},
		),
	}

	// Register all metrics
	if err := prometheus.Register(pc.eventsIngestedTotal); err != nil {
		return nil, fmt.Errorf("failed to register events_ingested_total: %w", err)
	}

	if err := prometheus.Register(pc.eventsPublishedTotal); err != nil {
		return nil, fmt.Errorf("failed to register events_published_total: %w", err)
	}

	if err := prometheus.Register(pc.eventsSubscribedTotal); err != nil {
		return nil, fmt.Errorf("failed to register events_subscribed_total: %w", err)
	}

	if err := prometheus.Register(&pc.errorsTotal); err != nil {
		return nil, fmt.Errorf("failed to register errors_total: %w", err)
	}

	if err := prometheus.Register(pc.ingestionDurationSeconds); err != nil {
		return nil, fmt.Errorf("failed to register event_ingestion_duration_seconds: %w", err)
	}

	if err := prometheus.Register(pc.publishDurationSeconds); err != nil {
		return nil, fmt.Errorf("failed to register event_publish_duration_seconds: %w", err)
	}

	if err := prometheus.Register(pc.activeSubscriptions); err != nil {
		return nil, fmt.Errorf("failed to register active_subscriptions: %w", err)
	}

	if err := prometheus.Register(pc.activeConsumers); err != nil {
		return nil, fmt.Errorf("failed to register active_consumers: %w", err)
	}

	if err := prometheus.Register(&pc.latencyHistogram); err != nil {
		return nil, fmt.Errorf("failed to register operation_latency_milliseconds: %w", err)
	}

	if err := prometheus.Register(&pc.subscriptionGaugeVec); err != nil {
		return nil, fmt.Errorf("failed to register subscriptions_active: %w", err)
	}

	return pc, nil
}

// RecordEventIngested increments the ingested events counter.
func (pc *PrometheusCollector) RecordEventIngested() {
	pc.eventsIngestedTotal.Inc()
}

// RecordEventPublished increments the published events counter.
func (pc *PrometheusCollector) RecordEventPublished() {
	pc.eventsPublishedTotal.Inc()
}

// RecordEventSubscribed increments the subscribed events counter.
func (pc *PrometheusCollector) RecordEventSubscribed() {
	pc.eventsSubscribedTotal.Inc()
}

// RecordError increments the error counter for a component and error type.
func (pc *PrometheusCollector) RecordError(component, errorType string) {
	pc.errorsTotal.WithLabelValues(component, errorType).Inc()
}

// RecordIngestionDuration records the duration of event ingestion.
func (pc *PrometheusCollector) RecordIngestionDuration(duration time.Duration) {
	pc.ingestionDurationSeconds.Observe(duration.Seconds())
}

// RecordPublishDuration records the duration of event publishing.
func (pc *PrometheusCollector) RecordPublishDuration(duration time.Duration) {
	pc.publishDurationSeconds.Observe(duration.Seconds())
}

// RecordActiveSubscription increments or decrements the active subscriptions gauge.
func (pc *PrometheusCollector) RecordActiveSubscription(delta int) {
	if delta > 0 {
		pc.activeSubscriptions.Add(float64(delta))
	} else {
		pc.activeSubscriptions.Sub(float64(-delta))
	}
}

// RecordActiveConsumer increments or decrements the active consumers gauge.
func (pc *PrometheusCollector) RecordActiveConsumer(delta int) {
	if delta > 0 {
		pc.activeConsumers.Add(float64(delta))
	} else {
		pc.activeConsumers.Sub(float64(-delta))
	}
}

// RecordIngestion records metrics for event ingestion.
func (pc *PrometheusCollector) RecordIngestion(eventCount int, durationMS float64) {
	// Record count
	for i := 0; i < eventCount; i++ {
		pc.eventsIngestedTotal.Inc()
	}
	// Record duration in seconds
	pc.ingestionDurationSeconds.Observe(durationMS / 1000.0)
}

// RecordPublish records metrics for event publishing.
func (pc *PrometheusCollector) RecordPublish(eventCount int, durationMS float64) {
	// Record count
	for i := 0; i < eventCount; i++ {
		pc.eventsPublishedTotal.Inc()
	}
	// Record duration in seconds
	pc.publishDurationSeconds.Observe(durationMS / 1000.0)
}

// RecordSubscription records metrics for event subscription.
func (pc *PrometheusCollector) RecordSubscription(consumerID, groupID string) {
	pc.subscriptionGaugeVec.WithLabelValues(consumerID, groupID).Inc()
}

// RecordLatency records the latency of an operation.
func (pc *PrometheusCollector) RecordLatency(operation string, durationMS float64) {
	pc.latencyHistogram.WithLabelValues(operation).Observe(durationMS)
}

// HTTPHandler returns the Prometheus metrics HTTP handler.
func (pc *PrometheusCollector) HTTPHandler() http.Handler {
	return promhttp.Handler()
}

// NewMetricsServer creates and returns an HTTP server for Prometheus metrics.
func NewMetricsServer(port int, path string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle(path, promhttp.Handler())

	return &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
