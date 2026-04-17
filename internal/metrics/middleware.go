package metrics

import (
	"net/http"
	"time"
)

type StatusRecorder struct {
	http.ResponseWriter
	Status int
}

func (r *StatusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		recorder := &StatusRecorder{ResponseWriter: w, Status: 200}

		next.ServeHTTP(recorder, r)

		duration := time.Since(start).Seconds()

		path := r.URL.Path

		HttpRequestsTotal.WithLabelValues(
			r.Method,
			path,
			http.StatusText(recorder.Status),
		).Inc()

		HttpRequestDuration.WithLabelValues(
			r.Method,
			path,
		).Observe(duration)
	})
}
