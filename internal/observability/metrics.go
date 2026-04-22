package observability

import (
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsOnce sync.Once

	holdRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "inventory_hold_requests_total",
			Help: "Total hold requests by status and error code.",
		},
		[]string{"status", "error_code"},
	)
	eventValidationFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "inventory_event_validation_failures_total",
			Help: "Total event validation failures by reason.",
		},
		[]string{"reason"},
	)
	stockConflicts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "inventory_stock_conflicts_total",
			Help: "Total stock conflicts by operation.",
		},
		[]string{"operation"},
	)
	confirmLatency = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "inventory_confirm_duration_seconds",
			Help:    "Confirmation operation latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
	)
	httpRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "inventory_http_requests_total",
			Help: "HTTP requests by route, method and status.",
		},
		[]string{"route", "method", "status_code"},
	)
)

func Register() {
	metricsOnce.Do(func() {
		prometheus.MustRegister(holdRequests, eventValidationFailures, stockConflicts, confirmLatency, httpRequests)
	})
}

func RecordHold(success bool, errorCode string) {
	status := "success"
	code := "NONE"
	if !success {
		status = "error"
		if errorCode != "" {
			code = errorCode
		}
	}
	holdRequests.WithLabelValues(status, code).Inc()
}

func RecordEventValidationFailure(reason string) {
	if reason == "" {
		reason = "unknown"
	}
	eventValidationFailures.WithLabelValues(reason).Inc()
}

func RecordStockConflict(operation string) {
	if operation == "" {
		operation = "unknown"
	}
	stockConflicts.WithLabelValues(operation).Inc()
}

func RecordConfirmDuration(d time.Duration) {
	confirmLatency.Observe(d.Seconds())
}

func RecordHTTPRequest(route, method string, statusCode int) {
	if route == "" {
		route = "unknown"
	}
	if method == "" {
		method = "UNKNOWN"
	}
	httpRequests.WithLabelValues(route, method, strconv.Itoa(statusCode)).Inc()
}
