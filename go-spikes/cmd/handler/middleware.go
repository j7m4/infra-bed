package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/infra-bed/go-spikes/pkg/metrics"
)

// HTTPMetricsMiddleware provides HTTP request metrics instrumentation
func HTTPMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Extract endpoint pattern from request path
		endpoint := extractEndpoint(r.URL.Path)
		
		// Track requests in flight
		metrics.HTTPRequestsInFlight.WithLabelValues(r.Method, endpoint).Inc()
		defer metrics.HTTPRequestsInFlight.WithLabelValues(r.Method, endpoint).Dec()

		// Wrap ResponseWriter to capture status code
		ww := &responseWrapper{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(ww, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(ww.statusCode)
		
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, endpoint, statusCode).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, endpoint).Observe(duration)
	})
}

// extractEndpoint converts URL paths to metric-friendly endpoint patterns
func extractEndpoint(path string) string {
	// Normalize common endpoint patterns
	switch {
	case strings.HasPrefix(path, "/cpu/fibonacci/"):
		return "/cpu/fibonacci/{n}"
	case strings.HasPrefix(path, "/config/feature/"):
		return "/config/feature/{feature}"
	case path == "/health":
		return "/health"
	case path == "/config":
		return "/config"
	case path == "/kafka/entity-repo":
		return "/kafka/entity-repo"
	case path == "/metrics":
		return "/metrics"
	default:
		return "unknown"
	}
}

// responseWrapper captures the HTTP status code
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWrapper) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}