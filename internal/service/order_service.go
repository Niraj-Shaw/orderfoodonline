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
	productService *ProductService
	orderRepo      repository.OrderRepository
	validator      promovalidator.ValidatorService
}

func NewOrderService(
	productService *ProductService,
	orderRepo repository.OrderRepository,
	validator promovalidator.ValidatorService,
) *OrderService {
	return &OrderService{
		productService: productService,
		orderRepo:      orderRepo,
		validator:      validator,
	}
}

// PlaceOrder validates input, resolves products (preserving item order),
// validates promo, assigns a UUID, persists, and returns the saved order.
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

	// Bulk validate existence → map[id]Product (lets us preserve item order)
	ids := make([]string, 0, len(req.Items))
	for _, it := range req.Items {
		ids = append(ids, it.ProductID)
	}
	prodMap, err := s.productService.ValidateProductsExist(ids)
	if err != nil {
		return nil, err // already ValidationError
	}

	// Resolve items/products in the same order as request
	resolvedItems := make([]models.OrderItem, 0, len(req.Items))
	resolvedProducts := make([]models.Product, 0, len(req.Items))
	for _, it := range req.Items {
		resolvedItems = append(resolvedItems, models.OrderItem{
			ProductID: it.ProductID,
			Quantity:  it.Quantity,
		})
		resolvedProducts = append(resolvedProducts, prodMap[it.ProductID])
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
