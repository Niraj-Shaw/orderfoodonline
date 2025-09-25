package service

import (
	"fmt"

	"github.com/Niraj-Shaw/orderfoodonline/internal/models"
	"github.com/Niraj-Shaw/orderfoodonline/internal/repository"
)

// ProductService handles product-related business logic.
type ProductService struct {
	repo repository.ProductRepository
}

func NewProductService(repo repository.ProductRepository) *ProductService {
	return &ProductService{repo: repo}
}

// List returns all products.
func (s *ProductService) GetAllProducts() ([]models.Product, error) {
	return s.repo.GetAll()
}

// Get returns a single product by ID (validation-style error if missing).
func (s *ProductService) GetProductByID(id string) (*models.Product, error) {
	if id == "" {
		return nil, NewValidationError("product ID cannot be empty")
	}
	p, err := s.repo.GetByID(id)
	if err != nil || p == nil {
		return nil, NewValidationError(fmt.Sprintf("product with ID %s not found", id))
	}
	return p, nil
}

// ValidateExistence checks that all IDs exist and returns a map[id]Product.
// Using a map lets callers (e.g., OrderService) preserve item ordering.
func (s *ProductService) ValidateProductsExist(ids []string) (map[string]models.Product, error) {
	if len(ids) == 0 {
		return nil, NewValidationError("no products provided")
	}
	out := make(map[string]models.Product, len(ids))
	for _, id := range ids {
		p, err := s.repo.GetByID(id)
		if err != nil || p == nil {
			return nil, NewValidationError(fmt.Sprintf("product with ID %s not found", id))
		}
		out[id] = *p
	}
	return out, nil
}
