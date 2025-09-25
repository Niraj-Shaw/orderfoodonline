// internal/service/order_service_test.go
package service

import (
	"errors"
	"testing"

	"github.com/Niraj-Shaw/orderfoodonline/internal/models"
	"github.com/Niraj-Shaw/orderfoodonline/internal/testutil"
)

func TestOrderService_PlaceOrder_TableDriven(t *testing.T) {
	type fields struct {
		productRepo *testutil.ProductRepoStub
		orderRepo   *testutil.OrderRepoStub
		validator   *testutil.ValidatorStub
	}
	type args struct {
		req models.OrderRequest
	}
	type want struct {
		orderNil        bool
		itemsLen        int
		productsLen     int
		errIsValidation bool
		errContains     string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "happy path with valid promo",
			fields: fields{
				productRepo: testutil.NewProductRepoStub(testutil.SeedProducts()),
				orderRepo:   testutil.NewOrderRepoStub(),
				validator:   &testutil.ValidatorStub{Valid: true},
			},
			args: args{
				req: models.OrderRequest{
					CouponCode: "HAPPYHRS",
					Items: []models.OrderItem{
						{ProductID: "1", Quantity: 2},
						{ProductID: "3", Quantity: 1},
					},
				},
			},
			want: want{orderNil: false, itemsLen: 2, productsLen: 2},
		},
		{
			name: "no promo still ok",
			fields: fields{
				productRepo: testutil.NewProductRepoStub(testutil.SeedProducts()),
				orderRepo:   testutil.NewOrderRepoStub(),
				validator:   &testutil.ValidatorStub{Valid: true}, // unused
			},
			args: args{
				req: models.OrderRequest{
					Items: []models.OrderItem{{ProductID: "2", Quantity: 1}},
				},
			},
			want: want{orderNil: false, itemsLen: 1, productsLen: 1},
		},
		{
			name: "invalid promo",
			fields: fields{
				productRepo: testutil.NewProductRepoStub(testutil.SeedProducts()),
				orderRepo:   testutil.NewOrderRepoStub(),
				validator:   &testutil.ValidatorStub{Valid: false},
			},
			args: args{
				req: models.OrderRequest{
					CouponCode: "BADCODE",
					Items:      []models.OrderItem{{ProductID: "1", Quantity: 1}},
				},
			},
			want: want{orderNil: true, errIsValidation: true, errContains: "invalid promo code"},
		},
		{
			name: "empty items",
			fields: fields{
				productRepo: testutil.NewProductRepoStub(testutil.SeedProducts()),
				orderRepo:   testutil.NewOrderRepoStub(),
				validator:   &testutil.ValidatorStub{Valid: true},
			},
			args: args{req: models.OrderRequest{}},
			want: want{orderNil: true, errIsValidation: true, errContains: "at least one item"},
		},
		{
			name: "missing productId",
			fields: fields{
				productRepo: testutil.NewProductRepoStub(testutil.SeedProducts()),
				orderRepo:   testutil.NewOrderRepoStub(),
				validator:   &testutil.ValidatorStub{Valid: true},
			},
			args: args{req: models.OrderRequest{Items: []models.OrderItem{{ProductID: "", Quantity: 1}}}},
			want: want{orderNil: true, errIsValidation: true, errContains: "productid is required"},
		},
		{
			name: "quantity <= 0",
			fields: fields{
				productRepo: testutil.NewProductRepoStub(testutil.SeedProducts()),
				orderRepo:   testutil.NewOrderRepoStub(),
				validator:   &testutil.ValidatorStub{Valid: true},
			},
			args: args{req: models.OrderRequest{Items: []models.OrderItem{{ProductID: "1", Quantity: 0}}}},
			want: want{orderNil: true, errIsValidation: true, errContains: "quantity must be > 0"},
		},
		{
			name: "product not found",
			fields: fields{
				productRepo: testutil.NewProductRepoStub(testutil.SeedProducts()),
				orderRepo:   testutil.NewOrderRepoStub(),
				validator:   &testutil.ValidatorStub{Valid: true},
			},
			args: args{req: models.OrderRequest{Items: []models.OrderItem{{ProductID: "999", Quantity: 1}}}},
			want: want{orderNil: true, errIsValidation: true, errContains: "not found"},
		},
		{
			name: "repo create error is non-validation error",
			fields: fields{
				productRepo: testutil.NewProductRepoStub(testutil.SeedProducts()),
				orderRepo:   &testutil.OrderRepoStub{Err: errors.New("db down")},
				validator:   &testutil.ValidatorStub{Valid: true},
			},
			args: args{req: models.OrderRequest{Items: []models.OrderItem{{ProductID: "1", Quantity: 1}}}},
			want: want{orderNil: true, errIsValidation: false, errContains: "failed to save order"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ps := NewProductService(tc.fields.productRepo)
			svc := NewOrderService(ps, tc.fields.orderRepo, tc.fields.validator)

			got, err := svc.PlaceOrder(tc.args.req)

			if tc.want.orderNil {
				if got != nil {
					t.Fatalf("expected nil order, got %+v", got)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got == nil || got.ID == "" {
					t.Fatalf("expected non-nil order with ID")
				}
				if len(got.Items) != tc.want.itemsLen || len(got.Products) != tc.want.productsLen {
					t.Fatalf("items/products mismatch got %d/%d want %d/%d",
						len(got.Items), len(got.Products), tc.want.itemsLen, tc.want.productsLen)
				}
			}

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
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
