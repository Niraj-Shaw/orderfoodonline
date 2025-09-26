// internal/testutil/stubs.go
package testutil

import (
	"errors"

	"github.com/Niraj-Shaw/orderfoodonline/internal/models"
	"github.com/Niraj-Shaw/orderfoodonline/internal/promovalidator"
	"github.com/Niraj-Shaw/orderfoodonline/internal/repository"
)

func SeedProducts() []models.Product {
	return []models.Product{
		{ID: "1", Name: "Chicken Waffle", Price: 12.99, Category: "Waffle"},
		{ID: "2", Name: "Belgian Waffle", Price: 9.99, Category: "Waffle"},
		{ID: "3", Name: "Caesar Salad", Price: 8.99, Category: "Salad"},
	}
}

// ContainsFold: simple case-insensitive substring check (no extra imports).
func ContainsFold(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			a, b := s[i+j], sub[j]
			if 'A' <= a && a <= 'Z' {
				a += 'a' - 'A'
			}
			if 'A' <= b && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

type ProductRepoStub struct {
	Products map[string]models.Product
	ErrAll   error // if set, GetAll() returns this error
	ErrByID  error // if set, GetByID() returns this error
}

func NewProductRepoStub(seed []models.Product) *ProductRepoStub {
	m := make(map[string]models.Product, len(seed))
	for _, p := range seed {
		m[p.ID] = p
	}
	return &ProductRepoStub{Products: m}
}

func (r *ProductRepoStub) GetAll() ([]models.Product, error) {
	if r.ErrAll != nil {
		return nil, r.ErrAll
	}
	out := make([]models.Product, 0, len(r.Products))
	for _, p := range r.Products {
		out = append(out, p)
	}
	return out, nil
}

func (r *ProductRepoStub) GetByID(id string) (*models.Product, error) {
	if r.ErrByID != nil {
		return nil, r.ErrByID
	}
	if p, ok := r.Products[id]; ok {
		cp := p
		return &cp, nil
	}
	return nil, nil
}

type OrderRepoStub struct {
	Stored *models.Order
	Err    error // if set, CreateOrder returns this error
}

func NewOrderRepoStub() *OrderRepoStub { return &OrderRepoStub{} }

func (r *OrderRepoStub) CreateOrder(o *models.Order) (*models.Order, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	cp := *o
	r.Stored = &cp
	return &cp, nil
}

func (r *OrderRepoStub) FindByID(id string) (*models.Order, error) {
	if r.Stored != nil && r.Stored.ID == id {
		cp := *r.Stored
		return &cp, nil
	}
	return nil, repository.ErrOrderNotFound
}

type ValidatorStub struct {
	Valid bool
	Err   error // if set, LoadCouponFiles returns this error
}

var _ promovalidator.ValidatorService = (*ValidatorStub)(nil)

func (v *ValidatorStub) LoadCouponFiles() error        { return v.Err }
func (v *ValidatorStub) ValidatePromoCode(string) bool { return v.Valid }

var (
	ErrRepoDown = errors.New("db down")
)
