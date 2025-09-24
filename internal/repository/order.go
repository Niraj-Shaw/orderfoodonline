package repository

import (
	"errors"

	"github.com/Niraj-Shaw/orderfoodonline/internal/models"
)

var (
	// Public, reusable errors
	ErrInvalidOrderID = errors.New("invalid order id (must be UUID)")
	ErrOrderExists    = errors.New("order already exists")
	ErrOrderNotFound  = errors.New("order not found")
)

// OrderRepository defines persistence for orders.
type OrderRepository interface {
	// CreateOrder persists a new order and returns the stored copy.
	CreateOrder(order *models.Order) (*models.Order, error)

	// FindByID retrieves an order by its ID.
	FindByID(id string) (*models.Order, error)
}
