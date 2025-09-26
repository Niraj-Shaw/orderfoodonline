package memory

import (
	"testing"

	"github.com/Niraj-Shaw/orderfoodonline/internal/models"
)

func TestProductRepo_Behavior(t *testing.T) {
	seed := []models.Product{
		{ID: "1", Name: "Chicken Waffle", Price: 12.99, Category: "Waffle"},
		{ID: "2", Name: "Caesar Salad", Price: 8.99, Category: "Salad"},
	}
	repo := NewProductRepo(seed)

	tests := []struct {
		name       string
		setup      func()
		action     func() (any, error)
		wantErr    error
		wantLength int
	}{
		{
			name: "GetAll returns seeded products",
			action: func() (any, error) {
				return repo.GetAll()
			},
			wantErr:    nil,
			wantLength: 2,
		},
		{
			name: "GetByID returns product",
			action: func() (any, error) {
				return repo.GetByID("1")
			},
			wantErr: nil,
		},
		{
			name: "GetByID not found",
			action: func() (any, error) {
				return repo.GetByID("99")
			},
			wantErr: ErrProductNotFound,
		},
		{
			name: "Create new product works",
			action: func() (any, error) {
				p := models.Product{ID: "3", Name: "Burger", Price: 10, Category: "FastFood"}
				return nil, repo.Create(p)
			},
			wantErr:    nil,
			wantLength: 3,
		},
		{
			name: "Create duplicate fails",
			action: func() (any, error) {
				p := models.Product{ID: "1", Name: "Duplicate", Price: 1, Category: "X"}
				return nil, repo.Create(p)
			},
			wantErr: ErrProductExists,
		},
		{
			name: "Update existing works",
			action: func() (any, error) {
				p := models.Product{ID: "1", Name: "Updated Chicken Waffle", Price: 12.99, Category: "Waffle"}
				return nil, repo.Update(p)
			},
			wantErr: nil,
		},
		{
			name: "Update missing fails",
			action: func() (any, error) {
				p := models.Product{ID: "99", Name: "Ghost", Price: 1}
				return nil, repo.Update(p)
			},
			wantErr: ErrProductNotFound,
		},
		{
			name: "Delete existing works",
			action: func() (any, error) {
				return nil, repo.Delete("1")
			},
			wantErr: nil,
		},
		{
			name: "Delete missing fails",
			action: func() (any, error) {
				return nil, repo.Delete("999")
			},
			wantErr: ErrProductNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			got, err := tt.action()
			if tt.wantErr != nil {
				if err == nil || err.Error() != tt.wantErr.Error() {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantLength > 0 {
				all, _ := repo.GetAll()
				if len(all) != tt.wantLength {
					t.Errorf("expected %d products, got %d", tt.wantLength, len(all))
				}
			}

			// sanity check for GetByID
			if tt.name == "GetByID returns product" {
				p, ok := got.(*models.Product)
				if !ok || p == nil {
					t.Errorf("expected product, got nil")
				}
				if p.ID != "1" {
					t.Errorf("expected ID=1, got %s", p.ID)
				}
			}
		})
	}
}
