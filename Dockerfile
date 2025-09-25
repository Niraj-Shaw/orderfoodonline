# syntax=docker/dockerfile:1.7

############################
# 1) Build stage
############################
ARG GO_VERSION=1.23.3
FROM golang:${GO_VERSION}-alpine AS builder

# Optional: if you prefer to let Go auto-fetch the exact toolchain, uncomment next line
# ENV GOTOOLCHAIN=auto

RUN apk add --no-cache git
WORKDIR /src

# Cache deps first
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags "-s -w" \
    -o /out/orderfoodonline ./cmd/server

############################
# 2) Runtime stage
############################
FROM alpine:3.20

# Minimal runtime deps (certs for HTTPS, curl for healthcheck)
RUN apk add --no-cache ca-certificates curl

# Non-root user
RUN adduser -D -u 10001 app
USER app

WORKDIR /app
COPY --from=builder /out/orderfoodonline /app/server

# Default envs (override at `docker run` if needed)
ENV PORT=8080 \
    API_KEY=apitest \
    COUPON_DIR=/app/data \
    LOG_LEVEL=info \
    GO_ENV=production

# You will bind-mount coupon files here at runtime
VOLUME ["/app/data"]

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -fsS http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/app/server"]