package models

// Product represents a food item
type Product struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Category string  `json:"category"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

// OrderRequest represents the request body for creating an order
type OrderRequest struct {
	CouponCode string      `json:"couponCode,omitempty"`
	Items      []OrderItem `json:"items"`
}

// Order represents a completed order
type Order struct {
	ID       string      `json:"id"`
	Items    []OrderItem `json:"items"`
	Products []Product   `json:"products"`
}

// ApiResponse represents a standard API response
type ApiResponse struct {
	Code    int    `json:"code"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status string `json:"status"`
}
