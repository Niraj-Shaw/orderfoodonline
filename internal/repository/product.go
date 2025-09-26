package repository

import "github.com/Niraj-Shaw/orderfoodonline/internal/models"

// Read-only (what the current API needs)
type ProductRepository interface {
	GetAll() ([]models.Product, error)
	GetByID(id string) (*models.Product, error)
}

// Optional write interface for future admin endpoints
type ProductWriter interface {
	Create(p models.Product) error
	Update(p models.Product) error
	Delete(id string) error
}

// If require to implement full CRUD in some component:
type ProductStore interface {
	ProductRepository
	ProductWriter
}
