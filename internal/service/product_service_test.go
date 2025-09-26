package service

import (
	"testing"

	"github.com/Niraj-Shaw/orderfoodonline/internal/testutil"
)

func TestProductService_List(t *testing.T) {
	t.Parallel()

	repo := testutil.NewProductRepoStub(testutil.SeedProducts())
	svc := NewProductService(repo)

	got, err := svc.GetAllProducts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 products, got %d", len(got))
	}
}

func TestProductService_Get(t *testing.T) {
	t.Parallel()

	type args struct{ id string }
	type want struct {
		productNil      bool
		productID       string
		errIsValidation bool
		errContains     string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{name: "success", args: args{id: "1"}, want: want{productNil: false, productID: "1"}},
		{name: "empty id", args: args{id: ""}, want: want{productNil: true, errIsValidation: true, errContains: "product id cannot be empty"}},
		{name: "missing", args: args{id: "999"}, want: want{productNil: true, errIsValidation: true, errContains: "not found"}},
	}

	repo := testutil.NewProductRepoStub(testutil.SeedProducts())
	svc := NewProductService(repo)

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := svc.GetProductByID(tc.args.id)

			if tc.want.productNil {
				if got != nil {
					t.Fatalf("expected nil product, got %+v", got)
				}
				if tc.want.errIsValidation && !IsValidationError(err) {
					t.Fatalf("expected ValidationError, got %T: %v", err, err)
				}
				if tc.want.errContains != "" && (err == nil || !testutil.ContainsFold(err.Error(), tc.want.errContains)) {
					t.Fatalf("expected error containing %q, got %v", tc.want.errContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil || got.ID != tc.want.productID {
				t.Fatalf("expected productID=%q, got %+v", tc.want.productID, got)
			}
		})
	}
}

func TestProductService_ValidateExistence(t *testing.T) {
	t.Parallel()

	type args struct{ ids []string }
	type want struct {
		mapLen          int
		errIsValidation bool
		errContains     string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{name: "all present", args: args{ids: []string{"1", "2"}}, want: want{mapLen: 2}},
		{name: "one missing", args: args{ids: []string{"1", "999"}}, want: want{errIsValidation: true, errContains: "not found"}},
		{name: "empty ids", args: args{ids: []string{}}, want: want{errIsValidation: true, errContains: "no products provided"}},
	}

	repo := testutil.NewProductRepoStub(testutil.SeedProducts())
	svc := NewProductService(repo)

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := svc.ValidateProductsExist(tc.args.ids)
			if tc.want.errContains != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.want.errContains)
				}
				if tc.want.errIsValidation && !IsValidationError(err) {
					t.Fatalf("expected ValidationError, got %T: %v", err, err)
				}
				if !testutil.ContainsFold(err.Error(), tc.want.errContains) {
					t.Fatalf("expected error to contain %q, got %q", tc.want.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tc.want.mapLen {
				t.Fatalf("expected %d entries, got %d", tc.want.mapLen, len(got))
			}
			// Light sanity check for "all present" path
			if tc.want.mapLen > 0 {
				for _, id := range tc.args.ids {
					if _, ok := got[id]; !ok {
						t.Fatalf("expected id %q in result", id)
					}
				}
			}
		})
	}
}
