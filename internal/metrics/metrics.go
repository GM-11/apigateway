package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	HttpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HttpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	AuthFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "auth_failures_total",
			Help: "Total failed auth attempts",
		},
	)

	RateLimitHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_hits_total",
			Help: "Rate limited requests",
		},
		[]string{"route"},
	)

	CircuitBreakerTrips = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "circuit_breaker_trips_total",
			Help: "Circuit breaker opened",
		},
	)
)

func Init() {
	prometheus.MustRegister(
		HttpRequestsTotal,
		HttpRequestDuration,
		AuthFailures,
		CircuitBreakerTrips,
		RateLimitHits,
	)
}
