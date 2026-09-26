# Stage 1: Build binary
FROM golang:1.22-alpine AS builder

# Install build dependencies (certificates, git, tzdata)
RUN apk add --no-cache ca-certificates tzdata git

WORKDIR /app

# Download dependencies first for layer caching
ENV GOPROXY=https://proxy.golang.org,direct
COPY go.mod go.sum ./
RUN go mod download

# Copy application source code
COPY . .

# Compile statically linked Go binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags '-static'" \
    -o /app/bin/api ./cmd/api/main.go

# Stage 2: Minimal runtime container
FROM alpine:3.20 AS runner

# Install essential runtime utilities for healthchecks and certificates
RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -g 10001 -S appgroup \
    && adduser -u 10001 -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder
COPY --from=builder --chown=appuser:appgroup /app/bin/api /usr/local/bin/api

# Switch to unprivileged user
USER appuser:appgroup

# Environment defaults
ENV PORT=8080 \
    GIN_MODE=release

# Expose HTTP port
EXPOSE 8080

# Health check using the /api/health endpoint
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 -O /dev/null http://localhost:8080/api/health || exit 1

ENTRYPOINT ["/usr/local/bin/api"]
