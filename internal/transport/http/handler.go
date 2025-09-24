package transporthttp

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Niraj-Shaw/orderfoodonline/internal/models"
	"github.com/Niraj-Shaw/orderfoodonline/internal/repository"
	"github.com/Niraj-Shaw/orderfoodonline/internal/service"
	"github.com/Niraj-Shaw/orderfoodonline/internal/util"
	"github.com/gorilla/mux"
)

type Handlers struct {
	productRepo  repository.ProductRepository
	orderService *service.OrderService
	logger       util.Logger
}

func NewHandlers(
	productRepo repository.ProductRepository,
	orderService *service.OrderService,
	logger util.Logger,
) *Handlers {
	return &Handlers{productRepo: productRepo, orderService: orderService, logger: logger}
}

// GET /healthz
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.sendJSON(w, http.StatusOK, models.HealthResponse{Status: "ok"})
}

// GET /api/product
func (h *Handlers) ListProducts(w http.ResponseWriter, r *http.Request) {
	ps, err := h.productRepo.GetAll()
	if err != nil {
		h.logger.Errorf("list products: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	h.sendJSON(w, http.StatusOK, ps)
}

// GET /api/product/{productId}
func (h *Handlers) GetProduct(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["productId"]

	// Optional: validate numeric because spec says int64
	if _, err := strconv.ParseInt(id, 10, 64); err != nil {
		h.sendError(w, http.StatusBadRequest, "error", "Invalid ID supplied")
		return
	}

	p, err := h.productRepo.GetByID(id)
	if err != nil || p == nil {
		h.sendError(w, http.StatusNotFound, "error", "Product not found")
		return
	}
	h.sendJSON(w, http.StatusOK, p)
}

// POST /api/order  (requires api_key via middleware)
func (h *Handlers) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	var req models.OrderReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "error", "Invalid input")
		return
	}

	order, err := h.orderService.PlaceOrder(req)
	if err != nil {
		// Keep it simple for now: treat service errors as validation issues per spec (422)
		h.sendError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}
	h.sendJSON(w, http.StatusOK, order)
}

// --- helpers ---

func (h *Handlers) sendJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.logger.Errorf("encode json: %v", err)
	}
}

func (h *Handlers) sendError(w http.ResponseWriter, status int, typ, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(models.ApiResponse{
		Code:    status,
		Type:    typ,
		Message: msg,
	})
}
