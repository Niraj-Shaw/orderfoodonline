package transporthttp

import (
	"encoding/json"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/Niraj-Shaw/orderfoodonline/internal/models"
	"github.com/Niraj-Shaw/orderfoodonline/internal/util"
)

// --- Logging ---

// LoggingMiddleware logs method, path, status, and duration.
func LoggingMiddleware(logger util.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)
			logger.Infof("%s %s %d %s", r.Method, r.RequestURI, wrapped.statusCode, time.Since(start))
		})
	}
}

// --- Recovery ---

// RecoveryMiddleware converts panics to 500 JSON and logs stack.
func RecoveryMiddleware(logger util.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Errorf("panic: %v\n%s", err, debug.Stack())
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(models.ApiResponse{
						Code:    http.StatusInternalServerError,
						Type:    "error",
						Message: "Internal server error",
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// --- CORS ---

// CORSMiddleware is permissive and handles preflight.
func CORSMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, api_key")
			w.Header().Set("Access-Control-Max-Age", "3600")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// --- API key ---

// APIKeyMiddleware validates header "api_key" for protected routes.
// Allows OPTIONS to pass for CORS preflight.
func APIKeyMiddleware(requiredAPIKey string, logger util.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			apiKey := r.Header.Get("api_key")
			if apiKey == "" {
				logger.Warnf("missing api_key: %s %s", r.Method, r.URL.Path)
				sendUnauthorized(w, "Missing API key")
				return
			}
			if apiKey != requiredAPIKey {
				logger.Warnf("invalid api_key: %s %s", r.Method, r.URL.Path)
				sendUnauthorized(w, "Invalid API key")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func sendUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(models.ApiResponse{
		Code:    http.StatusUnauthorized,
		Type:    "error",
		Message: message,
	})
}

// responseWriter captures status code for logging.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
