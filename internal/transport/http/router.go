package transporthttp

import (
	"context"
	"net/http"
	"time"

	"orderfoodonline/internal/config"
	"orderfoodonline/internal/repository"
	"orderfoodonline/internal/service"
	"orderfoodonline/internal/util"

	"github.com/gorilla/mux"
)

// Server wraps the HTTP server + handlers.
type Server struct {
	server   *http.Server
	logger   util.Logger
	handlers *Handlers
}

// NewServer composes router, handlers, and http.Server with sane timeouts.
func NewServer(
	cfg *config.Config,
	productRepo repository.ProductRepository,
	orderService *service.OrderService,
	logger util.Logger,
) *Server {
	h := NewHandlers(productRepo, orderService, logger)
	r := setupRouter(h, cfg, logger)

	s := &http.Server{
		Addr:         cfg.ServerAddr, // from config
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return &Server{server: s, logger: logger, handlers: h}
}

// setupRouter configures routes + global middleware.
func setupRouter(h *Handlers, cfg *config.Config, logger util.Logger) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	// Global middleware (keep simple versions for now)
	router.Use(LoggingMiddleware(logger))
	router.Use(RecoveryMiddleware(logger))
	router.Use(CORSMiddleware())

	// Health (no auth)
	router.HandleFunc("/healthz", h.HealthCheck).Methods(http.MethodGet)

	// API routes (OpenAPI server base is /api)
	api := router.PathPrefix("/api").Subrouter()

	// Product (public)
	api.HandleFunc("/product", h.ListProducts).Methods(http.MethodGet)
	api.HandleFunc("/product/{productId}", h.GetProduct).Methods(http.MethodGet)

	// Order (secured via api_key header)
	order := api.PathPrefix("").Subrouter()
	order.Use(APIKeyMiddleware(cfg.APIKey, logger)) // checks header: "api_key"
	order.HandleFunc("/order", h.PlaceOrder).Methods(http.MethodPost)

	return router
}

// Start begins serving (Addr was set from config).
func (s *Server) Start() error {
	s.logger.Infof("HTTP server listening on %s", s.server.Addr)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Stop gracefully shuts down with a timeout.
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}
