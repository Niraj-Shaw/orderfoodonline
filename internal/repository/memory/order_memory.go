package memory

import (
	"fmt"
	"sync"

	"github.com/Niraj-Shaw/orderfoodonline/internal/models"
	"github.com/Niraj-Shaw/orderfoodonline/internal/repository"
)

// orderMemoryRepository is a thread-safe in-memory store for orders.
type orderMemoryRepository struct {
	mutex  sync.RWMutex
	orders map[string]models.Order
}

// NewOrderRepo creates an empty in-memory order repository.
func NewOrderRepo() repository.OrderRepository {
	return &orderMemoryRepository{
		orders: make(map[string]models.Order),
	}
}

// Ensure it implements the interface
var _ repository.OrderRepository = (*orderMemoryRepository)(nil)

// CreateOrder adds a new order if it doesn’t already exist.
func (r *orderMemoryRepository) CreateOrder(order *models.Order) (*models.Order, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if order == nil || order.ID == "" {
		return nil, fmt.Errorf("order ID cannot be empty")
	}

	if _, exists := r.orders[order.ID]; exists {
		return nil, fmt.Errorf("order with ID %s already exists", order.ID)
	}

	r.orders[order.ID] = *order
	cp := r.orders[order.ID]
	return &cp, nil
}

// FindByID looks up an order by ID.
func (r *orderMemoryRepository) FindByID(id string) (*models.Order, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if order, exists := r.orders[id]; exists {
		cp := order
		return &cp, nil
	}
	return nil, fmt.Errorf("order with ID %s not found", id)
}
