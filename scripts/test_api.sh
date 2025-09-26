#!/usr/bin/env bash
set -euo pipefail

API_URL="http://localhost:8080"
API_KEY="apitest"

echo "🔍 Health check..."
curl -s -w "\n%{http_code}\n" "$API_URL/healthz"

echo -e "\n📦 Get all products..."
curl -s -w "\n%{http_code}\n" "$API_URL/api/product" \
  -H "api_key: $API_KEY"

echo -e "\n📦 Get product by ID (1)..."
curl -s -w "\n%{http_code}\n" "$API_URL/api/product/1" \
  -H "api_key: $API_KEY"

echo -e "\n🛒 Place order (no coupon)..."
curl -s -w "\n%{http_code}\n" -X POST "$API_URL/api/order" \
  -H "Content-Type: application/json" \
  -H "api_key: $API_KEY" \
  -d '{"items":[{"productId":"1","quantity":2}]}'

echo -e "\n🛒 Place order (with coupon)..."
curl -s -w "\n%{http_code}\n" -X POST "$API_URL/api/order" \
  -H "Content-Type: application/json" \
  -H "api_key: $API_KEY" \
  -d '{"items":[{"productId":"1","quantity":2}],"couponCode":"HAPPYHRS"}'