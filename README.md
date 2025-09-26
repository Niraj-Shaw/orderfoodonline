# Order Food Online API - Technical Challenge Solution

## 🎯 **Features**

	•	OpenAPI 3.1 compliant endpoints
	•	Clean layered architecture (handlers → services → repositories → models)
	•	Promo validation: 8–10 chars, must appear in 2+ coupon files
	•	Structured logging with slog (JSON in production)
	•	Repository pattern with in-memory implementations
	•	Unit tests for promo validator with edge cases
	•	Docker multi-stage build with health check
	•	Graceful shutdown with SIGINT/SIGTERM


## 🚀 **Quick Start**

### Prerequisites
	•	Go 1.21+
	•	Docker (for containerized run)
	•	Coupon files (couponbase1.gz, couponbase2.gz, couponbase3.gz)

### 1. Clone and Setup
```bash
# Clone your repository
git clone https://github.com/Niraj-Shaw/orderfoodonline.git
cd orderfoodonline

# Install dependencies
go mod tidy
```

### 2. Add Coupon Files
```bash
# Ensure the data directory exists
mkdir -p ./data

# Copy coupon files into ./data
cp /path/to/coupon/files/couponbase*.gz ./data/

# Verify
ls -lh ./data/couponbase*.gz
```

### 3. Run the Server
```bash
# Start server
go run cmd/server/main.go

# Expected output:
# server listening on :8080
# health:  http://:8080/healthz
# api:     http://:8080/api
```

### 4. Run with Docker
```bash
# Build image
docker build -t orderfoodonline .

# Run container
docker run -d \
  -p 8080:8080 \
  -e API_KEY=apitest \
  -e COUPON_DIR=/app/data \
  -v "$PWD/data:/app/data:ro" \
  --name orderfoodonline \
  orderfoodonline:latest
```

## 📡 **API Endpoints**

### Products
```bash
# List all products
GET /api/product

# Get specific product
GET /api/product/{productId}
```

### Orders
```bash
# Place order (requires api_key header)
POST /api/order
Content-Type: application/json
api_key: apitest

{
  "items": [
    {"productId": "1", "quantity": 2}
  ],
  "couponCode": "HAPPYHRS"
}
```

### Health
```bash
# Health check
GET /healthz
```

## 🏗️ **Architecture**

### Project Structure
```
orderfoodonline/
├── cmd/server/main.go              # Application entry point
├── internal/
│   ├── config/                     # Configuration management
│   ├── models/                     # Data models  
│   ├── repository/                 # Data access interfaces
│   │   └── memory/                 # In-memory implementations
│   ├── service/                    # Business logic
│   │   ├── product_service.go      # Product operations
│   │   └── order_service.go        # Order operations
│   ├── promovalidator/             # Promo code validation
│   ├── transport/http/             # HTTP transport layer
│   │   ├── router.go               # Routes and middleware
│   │   ├── handler.go              # HTTP handlers
│   │   └── middleware.go           # Middleware components
│   └── util/                       # Utilities
│       └── log.go                  # Structured logging (slog)
├── data/                          # Coupon files
├── scripts/                       # Utility scripts
└── bin/                          # Compiled binaries
```

### Architecture Layers
```
┌─────────────────────────────────────┐
│         HTTP Transport Layer        │  ← Handlers, Middleware, Routing
├─────────────────────────────────────┤
│          Service Layer              │  ← Business Logic (Product, Order)
├─────────────────────────────────────┤
│        Repository Layer             │  ← Data Access (Interfaces + Memory)
├─────────────────────────────────────┤
│         Models & Config             │  ← Data Structures & Configuration
└─────────────────────────────────────┘
```

### Dependency Flow
```
main.go → HTTP Server → Handlers → Services → Repositories → Models
```

## 🎫 **Promo Code Validation**

### Validation Rules
1. **Length**: 8-10 characters
2. **Format**: Alphanumeric only 
3. **Coverage**: Must appear in at least 2 files
4. **Files**: `couponbase1.gz`, `couponbase2.gz`, `couponbase3.gz`

## 🔧 **Configuration**

### Environment Variables
```bash
export PORT=8080                   # Server port (default: 8080)
export API_KEY=apitest             # API key (default: apitest)
export COUPON_DIR=./data           # Coupon files directory
export LOG_LEVEL=info              # Log level (debug, info, warn, error)
export GO_ENV=production           # Environment (enables JSON logging)
```

## 🧪 **Testing**

### Run All Tests
```bash
# Unit tests
go test ./... -v

# Promo validator tests (comprehensive)
go test ./internal/promovalidator -v

# API integration tests  
./scripts/test_api.sh

## 📊 **Logging**

```go
// Development output (text)
level=INFO msg="Order placed" order_id=uuid item_count=2

// Production output (JSON)
{"time":"2025-09-26T12:00:00Z","level":"INFO","msg":"Order placed","order_id":"uuid","item_count":2}
```

### Log Levels
- **DEBUG**: Detailed debugging information
- **INFO**: General operational messages  
- **WARN**: Warning conditions
- **ERROR**: Error conditions
- **FATAL**: Critical errors that cause exit

## 🐳 **Docker Deployment**

### Build and Run
```bash
# Build image
docker build -t orderfoodonline .

# Run container
docker run -p 8080:8080 orderfoodonline

```
## 📈 **API Examples**

### List Products
```bash
curl -X GET http://localhost:8080/api/product

# Response
[
  {
    "id": "1",
    "name": "Chicken Waffle", 
    "price": 12.99,
    "category": "Waffle"
  },
  ...
]
```

### Get Product by ID
```bash
curl -X GET http://localhost:8080/api/product/1

# Response
{
  "id": "1",
  "name": "Chicken Waffle",
  "price": 12.99,
  "category": "Waffle"
}
```

### Place Order
```bash
curl -X POST http://localhost:8080/api/order \
  -H "Content-Type: application/json" \
  -H "api_key: apitest" \
  -d '{
    "items": [
      {"productId": "1", "quantity": 2}
    ],
    "couponCode": "HAPPYHRS"
  }'

# Response  
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "items": [{"productId": "1", "quantity": 2}],
  "products": [{"id": "1", "name": "Chicken Waffle", ...}]
}
```

## 🔍 **Monitoring**

### Health Check
```bash
curl http://localhost:8080/healthz

# Response
{
  "status": "ok"
}
```

