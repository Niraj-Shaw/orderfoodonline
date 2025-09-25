// internal/transport/http/middleware_test.go
package transporthttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Niraj-Shaw/orderfoodonline/internal/models"
	"github.com/Niraj-Shaw/orderfoodonline/internal/util"
)

// helper: wraps a final handler with the given middleware chain in order
func chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// --- LoggingMiddleware ---

func TestLoggingMiddleware_PassesThroughStatusAndBody(t *testing.T) {
	logger := util.NewLogger()

	// final handler writes a specific status and body
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot) // 418
		_, _ = w.Write([]byte("brew"))
	})

	h := chain(final, LoggingMiddleware(logger))

	req := httptest.NewRequest(http.MethodGet, "/log", nil)
	rec := httptest.NewRecorder()

	start := time.Now()
	h.ServeHTTP(rec, req)
	dur := time.Since(start)

	if rec.Code != http.StatusTeapot {
		t.Fatalf("want 418, got %d", rec.Code)
	}
	if rec.Body.String() != "brew" {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
	// sanity: duration should be >=0 (we won't assert exact format)
	if dur < 0 {
		t.Fatalf("duration negative?")
	}
}

// --- RecoveryMiddleware ---

func TestRecoveryMiddleware_PanicTo500JSON(t *testing.T) {
	logger := util.NewLogger()

	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	h := chain(final, RecoveryMiddleware(logger))

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("want application/json, got %q", got)
	}
	var apiErr models.ApiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if apiErr.Code != http.StatusInternalServerError || apiErr.Type != "error" {
		t.Fatalf("unexpected error payload: %+v", apiErr)
	}
	if apiErr.Message == "" {
		t.Fatalf("want non-empty Message")
	}
}

// --- CORSMiddleware ---

func TestCORSMiddleware_OptionsShortCircuitAndHeaders(t *testing.T) {
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// should not be reached on OPTIONS
		w.WriteHeader(http.StatusOK)
	})

	h := chain(final, CORSMiddleware())

	// Preflight
	req := httptest.NewRequest(http.MethodOptions, "/any", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS want 204, got %d", rec.Code)
	}

	// CORS headers present?
	hdr := rec.Header()
	if hdr.Get("Access-Control-Allow-Origin") == "" ||
		hdr.Get("Access-Control-Allow-Methods") == "" ||
		hdr.Get("Access-Control-Allow-Headers") == "" ||
		hdr.Get("Access-Control-Max-Age") == "" {
		t.Fatalf("missing CORS headers: %+v", hdr)
	}

	// Regular GET passes through and still has headers
	req2 := httptest.NewRequest(http.MethodGet, "/any", nil)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK { // final wrote 200 on GET
		t.Fatalf("GET want 200, got %d", rec2.Code)
	}
	if rec2.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatalf("GET missing CORS headers")
	}
}

// --- APIKeyMiddleware ---

func TestAPIKeyMiddleware_Table(t *testing.T) {
	logger := util.NewLogger()
	const required = "apitest"

	// final handler (should be reached only when authorized or when OPTIONS)
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	h := chain(final, APIKeyMiddleware(required, logger))

	tests := []struct {
		name       string
		method     string
		apiKey     string
		wantStatus int
	}{
		{
			name:       "OPTIONS bypasses auth",
			method:     http.MethodOptions,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "missing key",
			method:     http.MethodGet,
			apiKey:     "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid key",
			method:     http.MethodPost,
			apiKey:     "wrong",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "valid key",
			method:     http.MethodPost,
			apiKey:     "apitest",
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/secure", nil)
			if tc.apiKey != "" {
				req.Header.Set("api_key", tc.apiKey)
			}
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("want %d, got %d, body=%s", tc.wantStatus, rec.Code, rec.Body.String())
			}
			if tc.method == http.MethodOptions && rec.Code != http.StatusNoContent {
				t.Fatalf("preflight should be 204")
			}
			if tc.wantStatus == http.StatusUnauthorized {
				// Should be JSON error
				if rec.Header().Get("Content-Type") != "application/json" {
					t.Fatalf("want JSON content-type on 401")
				}
				var apiErr models.ApiResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
					t.Fatalf("decode 401 json: %v", err)
				}
				if apiErr.Code != http.StatusUnauthorized || apiErr.Message == "" {
					t.Fatalf("unexpected 401 payload: %+v", apiErr)
				}
			}
		})
	}
}
