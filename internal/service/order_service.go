package service

import (
	"errors"

	"github.com/Niraj-Shaw/orderfoodonline/internal/models"
	"github.com/Niraj-Shaw/orderfoodonline/internal/promovalidator"
	"github.com/Niraj-Shaw/orderfoodonline/internal/repository"
	"github.com/google/uuid"
)

// OrderService defines the interface for order-related business logic.
type OrderService interface {
	CreateOrder(req models.OrderRequest) (*models.Order, error)
	GetOrderByID(id string) (*models.Order, error)
	GetProducts() ([]models.Product, error)
}

// orderService implements the OrderService interface.
type orderService struct {
	productRepo repository.ProductRepository
	orderRepo   repository.OrderRepository
	promoVal    promovalidator.ValidatorService
}

// NewOrderService creates a new OrderService.
func NewOrderService(productRepo repository.ProductRepository, orderRepo repository.OrderRepository, promoVal promovalidator.ValidatorService) OrderService {
	return &orderService{
		productRepo: productRepo,
		orderRepo:   orderRepo,
		promoVal:    promoVal,
	}
}

// CreateOrder handles the business logic for creating a new order.
func (s *orderService) CreateOrder(req models.OrderRequest) (*models.Order, error) {
	if len(req.Items) == 0 {
		return nil, errors.New("order must contain at least one item")
	}

	// In a real application, you would have more complex logic here, such as:
	// - Checking product availability and stock
	// - Calculating the total price
	// - Applying the coupon discount

	if req.CouponCode != "" {
		if !s.promoVal.ValidatePromoCode(req.CouponCode) {
			return nil, errors.New("invalid coupon code")
		}
	}

	products, err := s.productRepo.GetProducts()
	if err != nil {
		return nil, err
	}

	newOrder := &models.Order{
		ID:       uuid.New().String(),
		Items:    req.Items,
		Products: products, // Simplified: In reality, you'd match items to products
	}

	if err := s.orderRepo.CreateOrder(*newOrder); err != nil {
		return nil, err
	}

	return newOrder, nil
}

// GetOrderByID retrieves an order by its ID.
func (s *orderService) GetOrderByID(id string) (*models.Order, error) {
	return s.orderRepo.GetOrderByID(id)
}

// GetProducts retrieves all products.
func (s *orderService) GetProducts() ([]models.Product, error) {
	return s.productRepo.GetProducts()
}