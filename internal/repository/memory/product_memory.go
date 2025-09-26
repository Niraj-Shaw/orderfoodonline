package memory

import (
	"errors"
	"sync"

	"github.com/Niraj-Shaw/orderfoodonline/internal/models"
	"github.com/Niraj-Shaw/orderfoodonline/internal/repository"
)

var (
	ErrProductNotFound  = errors.New("product not found")
	ErrProductExists    = errors.New("product already exists")
	ErrInvalidProductID = errors.New("invalid product id")
)

type ProductRepo struct {
	mu       sync.RWMutex
	products map[string]models.Product
}

// NewProductRepo seeds the repo with initial products.
func NewProductRepo(seed []models.Product) *ProductRepo {
	m := make(map[string]models.Product, len(seed))
	for _, p := range seed {
		m[p.ID] = p
	}
	return &ProductRepo{products: m}
}

var _ repository.ProductRepository = (*ProductRepo)(nil)
var _ repository.ProductWriter = (*ProductRepo)(nil) // Optional

func (r *ProductRepo) GetAll() ([]models.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]models.Product, 0, len(r.products))
	for _, p := range r.products {
		out = append(out, p)
	}
	return out, nil
}

func (r *ProductRepo) GetByID(id string) (*models.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.products[id]
	if !ok {
		return nil, ErrProductNotFound
	}
	cp := p
	return &cp, nil
}

// --- Optional CRUD ---

func (r *ProductRepo) Create(p models.Product) error {
	if p.ID == "" {
		return ErrInvalidProductID
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.products[p.ID]; exists {
		return ErrProductExists
	}
	r.products[p.ID] = p
	return nil
}

func (r *ProductRepo) Update(p models.Product) error {
	if p.ID == "" {
		return ErrInvalidProductID
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.products[p.ID]; !ok {
		return ErrProductNotFound
	}
	r.products[p.ID] = p
	return nil
}

func (r *ProductRepo) Delete(id string) error {
	if id == "" {
		return ErrInvalidProductID
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.products[id]; !ok {
		return ErrProductNotFound
	}
	delete(r.products, id)
	return nil
}
