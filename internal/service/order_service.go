// internal/service/order_service.go
package service

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/Niraj-Shaw/orderfoodonline/internal/models"
	"github.com/Niraj-Shaw/orderfoodonline/internal/promovalidator"
	"github.com/Niraj-Shaw/orderfoodonline/internal/repository"
)

// ----- validation error type & helper -----

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

func NewValidationError(msg string) *ValidationError { return &ValidationError{Message: msg} }

func IsValidationError(err error) bool {
	var v *ValidationError
	return errors.As(err, &v)
}

// ----- service -----

// OrderService handles business logic for order operations.
type OrderService struct {
	productRepo repository.ProductRepository
	orderRepo   repository.OrderRepository
	validator   promovalidator.ValidatorService
}

// NewOrderService constructs the service.
func NewOrderService(
	productRepo repository.ProductRepository,
	orderRepo repository.OrderRepository,
	validator promovalidator.ValidatorService,
) *OrderService {
	return &OrderService{
		productRepo: productRepo,
		orderRepo:   orderRepo,
		validator:   validator,
	}
}

// PlaceOrder validates input, resolves products, validates promo (if any),
// assigns a UUID, persists the order, and returns the saved copy.
func (s *OrderService) PlaceOrder(req models.OrderRequest) (*models.Order, error) {
	// Basic request validation
	if len(req.Items) == 0 {
		return nil, NewValidationError("order must contain at least one item")
	}
	for i, it := range req.Items {
		if it.ProductID == "" {
			return nil, NewValidationError(fmt.Sprintf("item %d: productId is required", i+1))
		}
		if it.Quantity <= 0 {
			return nil, NewValidationError(fmt.Sprintf("item %d: quantity must be > 0", i+1))
		}
	}

	// Promo validation (case-sensitive) if provided
	if req.CouponCode != "" {
		if !s.validator.ValidatePromoCode(req.CouponCode) {
			return nil, NewValidationError("invalid promo code")
		}
	}

	// Resolve products
	resolvedItems := make([]models.OrderItem, 0, len(req.Items))
	resolvedProducts := make([]models.Product, 0, len(req.Items))

	for _, it := range req.Items {
		p, err := s.productRepo.GetByID(it.ProductID)
		if err != nil || p == nil {
			return nil, NewValidationError(fmt.Sprintf("product with ID %s not found", it.ProductID))
		}
		resolvedItems = append(resolvedItems, models.OrderItem{
			ProductID: it.ProductID,
			Quantity:  it.Quantity,
		})
		resolvedProducts = append(resolvedProducts, *p)
	}

	// Build order with UUID
	order := &models.Order{
		ID:       uuid.New().String(),
		Items:    resolvedItems,
		Products: resolvedProducts,
	}

	// Persist
	saved, err := s.orderRepo.CreateOrder(order)
	if err != nil {
		return nil, fmt.Errorf("failed to save order: %w", err)
	}
	return saved, nil
}
