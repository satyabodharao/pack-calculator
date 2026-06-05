package api

import (
	"log/slog"
	"net/http"
	"time"
)

// NewRouter builds the HTTP handler tree: API endpoints plus static UI serving,
// wrapped with a request-logging middleware.
//
// webDir is the path to the directory containing the static UI (index.html).
func NewRouter(h *Handler, webDir string, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	// API endpoints.
	mux.HandleFunc("/api/pack-sizes", h.PackSizes)
	mux.HandleFunc("/api/calculate", h.Calculate)
	mux.HandleFunc("/healthz", h.Health)

	// Static UI. The file server handles "/" and any static assets in webDir.
	mux.Handle("/", http.FileServer(http.Dir(webDir)))

	return loggingMiddleware(logger)(mux)
}

// statusRecorder captures the response status code so the logging middleware can
// report it.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// loggingMiddleware logs one structured line per request with method, path,
// status and latency. Critical for observing the service in any environment.
func loggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}
