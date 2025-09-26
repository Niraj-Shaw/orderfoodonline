package transporthttp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"github.com/Niraj-Shaw/orderfoodonline/internal/config"
	"github.com/Niraj-Shaw/orderfoodonline/internal/models"
	"github.com/Niraj-Shaw/orderfoodonline/internal/service"
	"github.com/Niraj-Shaw/orderfoodonline/internal/testutil"
	"github.com/Niraj-Shaw/orderfoodonline/internal/util"
)

// shared helper
func setupHandlers(validatorValid bool) (*Handlers, *config.Config, util.Logger) {
	prodRepo := testutil.NewProductRepoStub(testutil.SeedProducts())
	prodSvc := service.NewProductService(prodRepo)

	ordRepo := testutil.NewOrderRepoStub()
	validator := &testutil.ValidatorStub{Valid: validatorValid}

	ordSvc := service.NewOrderService(prodSvc, ordRepo, validator)
	logger := util.NewLogger()
	cfg := &config.Config{APIKey: "apitest"}

	return NewHandlers(prodSvc, ordSvc, logger), cfg, logger
}

func TestHandlers(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		target     string
		body       string
		apiKey     string
		validator  bool
		wantStatus int
	}{
		{
			name:       "ListProducts OK",
			method:     http.MethodGet,
			target:     "/api/product",
			wantStatus: http.StatusOK,
		},
		{
			name:       "GetProduct OK",
			method:     http.MethodGet,
			target:     "/api/product/1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "GetProduct invalid ID",
			method:     http.MethodGet,
			target:     "/api/product/abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "GetProduct not found",
			method:     http.MethodGet,
			target:     "/api/product/999",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "PlaceOrder missing API key",
			method:     http.MethodPost,
			target:     "/api/order",
			body:       `{"items":[{"productId":"1","quantity":1}]}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "PlaceOrder bad JSON",
			method:     http.MethodPost,
			target:     "/api/order",
			body:       `{"bad json"`,
			apiKey:     "apitest",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "PlaceOrder invalid promo code",
			method:     http.MethodPost,
			target:     "/api/order",
			body:       `{"items":[{"productId":"1","quantity":1}],"couponCode":"BAD"}`,
			apiKey:     "apitest",
			validator:  false, // force validator to reject
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "PlaceOrder success",
			method:     http.MethodPost,
			target:     "/api/order",
			body:       `{"items":[{"productId":"1","quantity":2}]}`,
			apiKey:     "apitest",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, cfg, logger := setupHandlers(tt.validator)

			// router per test
			r := mux.NewRouter()
			api := r.PathPrefix("/api").Subrouter()

			// public routes
			api.HandleFunc("/product", h.ListProducts).Methods(http.MethodGet)
			api.HandleFunc("/product/{productId}", h.GetProduct).Methods(http.MethodGet)

			// secured routes
			secured := api.PathPrefix("").Subrouter()
			secured.Use(APIKeyMiddleware(cfg.APIKey, logger))
			secured.HandleFunc("/order", h.PlaceOrder).Methods(http.MethodPost)

			// build request
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.target, bytes.NewBufferString(tt.body))
			} else {
				req = httptest.NewRequest(tt.method, tt.target, nil)
			}
			if tt.apiKey != "" {
				req.Header.Set("api_key", tt.apiKey)
			}
			rec := httptest.NewRecorder()

			// serve
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("want %d, got %d. Body=%s", tt.wantStatus, rec.Code, rec.Body.String())
			}

			// extra sanity checks on 200s
			if tt.wantStatus == http.StatusOK {
				if tt.target == "/api/product" {
					var got []models.Product
					_ = json.Unmarshal(rec.Body.Bytes(), &got)
					if len(got) == 0 {
						t.Errorf("expected non-empty product list")
					}
				}
				if tt.target == "/api/order" && tt.method == http.MethodPost {
					var got models.Order
					_ = json.Unmarshal(rec.Body.Bytes(), &got)
					if got.ID == "" {
						t.Errorf("expected order ID, got empty")
					}
				}
			}
		})
	}
}
