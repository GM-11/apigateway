package metrics

import (
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		recorder := &statusRecorder{ResponseWriter: w, status: 200}

		next.ServeHTTP(recorder, r)

		duration := time.Since(start).Seconds()

		path := r.URL.Path

		HttpRequestsTotal.WithLabelValues(
			r.Method,
			path,
			http.StatusText(recorder.status),
		).Inc()

		HttpRequestDuration.WithLabelValues(
			r.Method,
			path,
		).Observe(duration)
	})
}
