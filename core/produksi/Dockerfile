# ============================================
# Produksi Service Dockerfile
# Build context: servfor/ root
# Usage: docker build -f core/produksi/Dockerfile .
# ============================================

# Stage 1: Build Go Binary
FROM golang:1.24-alpine AS go-builder
WORKDIR /build

RUN apk add --no-cache git ca-certificates tzdata

COPY core/produksi/go.mod core/produksi/go.sum ./
RUN go mod download

COPY core/produksi/ .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o produksi-server .

# Stage 2: Runtime
FROM alpine:3.18
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Jakarta

WORKDIR /app
COPY --from=go-builder /build/produksi-server .
COPY core/produksi/web ./web

EXPOSE 3001

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

CMD ["/app/produksi-server"]
