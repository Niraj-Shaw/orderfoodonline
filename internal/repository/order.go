package repository

import "github.com/Niraj-Shaw/orderfoodonline/internal/models"

// OrderRepository defines the interface for order data operations.
type OrderRepository interface {
	CreateOrder(order models.Order) error
	GetOrderByID(id string) (*models.Order, error)
}