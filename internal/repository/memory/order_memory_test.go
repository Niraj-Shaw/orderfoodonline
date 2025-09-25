package memory

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Niraj-Shaw/orderfoodonline/internal/models"
	"github.com/Niraj-Shaw/orderfoodonline/internal/repository"
)

// Helper to build a repo for each test
func newRepo(t *testing.T) repository.OrderRepository {
	t.Helper()
	return NewOrderRepo()
}

func TestCreateOrder_SuccessAndCopySemantics(t *testing.T) {
	t.Parallel()

	repo := newRepo(t)

	// Build a valid order with UUID
	id := uuid.New().String()
	input := &models.Order{
		ID:       id,
		Items:    []models.OrderItem{},
		Products: []models.Product{},
	}

	// Create
	saved, err := repo.CreateOrder(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved == nil || saved.ID != id {
		t.Fatalf("expected saved order with id=%s, got %+v", id, saved)
	}

	// Mutate the original after CreateOrder; repo should have stored a copy
	input.Items = append(input.Items, models.OrderItem{ProductID: "1", Quantity: 99})

	// Fetch back and ensure stored value was not affected by post-create mutation
	got, err := repo.FindByID(id)
	if err != nil {
		t.Fatalf("FindByID unexpected error: %v", err)
	}
	if len(got.Items) != 0 {
		t.Fatalf("repo did not store a copy; expected 0 items, got %d", len(got.Items))
	}
}

func TestCreateOrder_Errors(t *testing.T) {
	t.Parallel()

	repo := newRepo(t)

	tests := []struct {
		name      string
		order     *models.Order
		wantError string // substring match
		wantIs    error  // optional: sentinel
	}{
		{
			name:      "nil order",
			order:     nil,
			wantError: "order ID cannot be empty",
		},
		{
			name:      "empty id",
			order:     &models.Order{ID: ""},
			wantError: "order ID cannot be empty",
		},
		{
			name:   "invalid uuid",
			order:  &models.Order{ID: "not-a-uuid"},
			wantIs: repository.ErrInvalidOrderID,
		},
		{
			name: "duplicate id",
			order: &models.Order{
				ID: uuid.New().String(),
			},
			// second create with same id should fail with a "already exists" message
			wantError: "already exists",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "duplicate id" {
				// seed the first successful create
				if _, err := repo.CreateOrder(tc.order); err != nil {
					t.Fatalf("seed create failed: %v", err)
				}
			}

			_, err := repo.CreateOrder(tc.order)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			if tc.wantIs != nil && !errors.Is(err, tc.wantIs) {
				t.Fatalf("want errors.Is(err, %v)=true; got err=%v", tc.wantIs, err)
			}
			if tc.wantError != "" && !containsFold(err.Error(), tc.wantError) {
				t.Fatalf("want error containing %q, got %q", tc.wantError, err.Error())
			}
		})
	}
}

func TestFindByID_Behavior(t *testing.T) {
	t.Parallel()

	repo := newRepo(t)

	validID := uuid.New().String()
	if _, err := repo.CreateOrder(&models.Order{ID: validID}); err != nil {
		t.Fatalf("seed create failed: %v", err)
	}

	tests := []struct {
		name      string
		id        string
		wantIs    error  // expected sentinel (or nil for success)
		wantError string // substring (optional)
	}{
		{
			name:   "success",
			id:     validID,
			wantIs: nil,
		},
		{
			name:   "invalid uuid",
			id:     "bad-id",
			wantIs: repository.ErrInvalidOrderID,
		},
		{
			name:   "not found",
			id:     uuid.New().String(),
			wantIs: repository.ErrOrderNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			o, err := repo.FindByID(tc.id)

			if tc.wantIs == nil {
				// success case
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if o == nil || o.ID != tc.id {
					t.Fatalf("expected order with id=%s, got %+v", tc.id, o)
				}
				return
			}

			// error cases
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.Is(err, tc.wantIs) {
				t.Fatalf("want errors.Is(err, %v)=true; got err=%v", tc.wantIs, err)
			}
			if tc.wantError != "" && !containsFold(err.Error(), tc.wantError) {
				t.Fatalf("want error containing %q, got %q", tc.wantError, err.Error())
			}
		})
	}
}

// containsFold: simple case-insensitive substring check
func containsFold(s, sub string) bool {
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
