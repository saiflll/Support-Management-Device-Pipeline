# ============================================
# Stage 1: Build
# ============================================
ARG GOLANG_VERSION=1.24-alpine
FROM golang:${GOLANG_VERSION} AS builder

WORKDIR /build

# Install dependencies (with retry for temporary network errors)
RUN apk add --no-cache git ca-certificates tzdata || \
    (sleep 5 && apk add --no-cache git ca-certificates tzdata) || \
    (sleep 10 && apk add --no-cache git ca-certificates tzdata)

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o forwarder-server main.go

# ============================================
# Stage 2: Runtime
# ============================================
FROM alpine:3.18

# Install runtime dependencies (with retry for temporary network errors)
RUN apk add --no-cache ca-certificates tzdata || \
    (sleep 5 && apk add --no-cache ca-certificates tzdata) || \
    (sleep 10 && apk add --no-cache ca-certificates tzdata)

# Set timezone
ENV TZ=Asia/Jakarta

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/forwarder-server .

# Copy required directories (templates for forwarder UI)
COPY --from=builder /build/internal/forwarder/*.html ./internal/forwarder/

# Expose port
EXPOSE 8888

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:8888/forwarder/status || exit 1

# Run as non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

# Run the application
CMD ["/app/forwarder-server"]
