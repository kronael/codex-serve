package main

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// requests_total counts total API requests by endpoint, method, and status
	requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "requests_total",
			Help: "Total number of API requests",
		},
		[]string{"endpoint", "method", "status"},
	)

	// request_duration tracks request latency by endpoint
	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint", "method"},
	)

	// active_sessions tracks current WebSocket sessions
	activeSessions = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_sessions",
			Help: "Current number of active WebSocket sessions",
		},
	)

	// tokens_total counts tokens processed by model and type
	tokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tokens_total",
			Help: "Total number of tokens processed",
		},
		[]string{"model", "type"},
	)
)

// MetricsCollector wraps Prometheus metrics
type MetricsCollector struct{}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

// RecordRequest records a request metric
func (m *MetricsCollector) RecordRequest(endpoint, model string, duration time.Duration) {
	requestsTotal.WithLabelValues(endpoint, "POST", "200").Inc()
	requestDuration.WithLabelValues(endpoint, "POST").Observe(duration.Seconds())
}

// RecordRequestWithStatus records a request with specific status
func (m *MetricsCollector) RecordRequestWithStatus(endpoint, method string, status int) {
	requestsTotal.WithLabelValues(endpoint, method, strconv.Itoa(status)).Inc()
}

// RecordTokens records token usage
func (m *MetricsCollector) RecordTokens(model string, input, output int) {
	tokensTotal.WithLabelValues(model, "input").Add(float64(input))
	tokensTotal.WithLabelValues(model, "output").Add(float64(output))
}

// IncrementSessions increments active session count
func (m *MetricsCollector) IncrementSessions() {
	activeSessions.Inc()
}

// DecrementSessions decrements active session count
func (m *MetricsCollector) DecrementSessions() {
	activeSessions.Dec()
}
