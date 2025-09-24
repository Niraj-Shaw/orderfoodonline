# syntax=docker/dockerfile:1.7

############################
# 1) Build stage
############################
FROM golang:1.22-alpine AS builder

# install toolchain for private modules, etc.
RUN apk add --no-cache git

WORKDIR /app

# cache deps
COPY go.mod go.sum ./
RUN go mod download

# copy source
COPY . .

# build static binary
RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags "-s -w" \
    -o /out/orderfoodonline ./cmd/server

############################
# 2) Runtime stage
############################
FROM alpine:3.20

# add non-root user
RUN adduser -D -u 10001 app

# minimal runtime deps (curl for healthcheck, certs for HTTPS)
RUN apk add --no-cache ca-certificates curl

# config (override at runtime as needed)
ENV SERVER_ADDR=":8080" \
    API_KEY="apitest" \
    COUPON_DIR="/data"

WORKDIR /
COPY --from=builder /out/orderfoodonline /orderfoodonline

# where you will MOUNT your big coupon files at runtime
VOLUME ["/data"]

EXPOSE 8080
USER app

# healthcheck hits your app's /healthz
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -fsS http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/orderfoodonline"]