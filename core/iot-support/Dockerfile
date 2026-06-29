# ============================================================
# IoT Support: Combined Forwarder + OTA Server
# Build context: servfor/ root
# Usage: docker build -f core/iot-support/Dockerfile .
# ============================================================

# ============================================================
# Stage 1: Build Svelte UI (untuk OTA Server)
# ============================================================
FROM node:22-alpine AS svelte-builder
WORKDIR /ui
COPY view/package*.json ./
RUN npm ci
COPY view .
RUN npm run build

# ============================================================
# Stage 2: Build Forwarder Binary
# ============================================================
FROM golang:1.24-alpine AS forwarder-builder
WORKDIR /build
RUN apk add --no-cache git ca-certificates tzdata || \
    (sleep 5 && apk add --no-cache git ca-certificates tzdata)
COPY core/forward/go.mod core/forward/go.sum ./
COPY core/forward/vendor ./vendor
COPY core/forward/ .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o forwarder-server main.go

# ============================================================
# Stage 3: Build OTA Binary
# ============================================================
FROM golang:1.24-alpine AS ota-builder
WORKDIR /build
RUN apk add --no-cache git ca-certificates tzdata || \
    (sleep 5 && apk add --no-cache git ca-certificates tzdata)
COPY core/ota/go.mod core/ota/go.sum ./
COPY core/ota/vendor ./vendor
COPY core/ota/ .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o ota-server .

# ============================================================
# Stage 4: Runtime - Gabungkan keduanya dalam satu container
# ============================================================
FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata || \
    (sleep 5 && apk add --no-cache ca-certificates tzdata)

ENV TZ=Asia/Jakarta

WORKDIR /app

# Salin binary forwarder
COPY --from=forwarder-builder /build/forwarder-server .
COPY --from=forwarder-builder /build/internal/forwarder/*.html ./internal/forwarder/

# Salin binary OTA + assets-nya
COPY --from=ota-builder /build/ota-server .
COPY --from=ota-builder /build/static ./static
COPY --from=svelte-builder /ui/build ./web

# Buat folder upload OTA
RUN mkdir -p /app/static/uploads

# Salin entrypoint script
COPY core/iot-support/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# Expose port forwarder (8888) dan ota (9999)
EXPOSE 8888 9999

# Health check: pastikan forwarder berjalan (port utama)
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:8888/forwarder/status || exit 1

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

ENTRYPOINT ["/app/entrypoint.sh"]
