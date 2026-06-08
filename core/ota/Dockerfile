
ARG GOLANG_VERSION=1.24-alpine

# ============================================
# Stage 1: Build Svelte Dashboard
# ============================================
FROM node:22-alpine AS svelte-builder
WORKDIR /ui
COPY view/package*.json ./
RUN npm ci
COPY view .
RUN npm run build

# ============================================
# Stage 2: Build Go OTA Binary
# ============================================
FROM golang:${GOLANG_VERSION} AS go-builder

WORKDIR /build

# Install dependencies (with retry for temporary network errors)
RUN apk add --no-cache git ca-certificates tzdata || \
    (sleep 5 && apk add --no-cache git ca-certificates tzdata) || \
    (sleep 10 && apk add --no-cache git ca-certificates tzdata)

COPY core/ota/go.mod core/ota/go.sum ./
COPY core/ota/vendor ./vendor

COPY core/ota/ .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o ota-server .

# ============================================
# Stage 3: Runtime
# ============================================
FROM alpine:3.18

# Install runtime dependencies (with retry for temporary network errors)
RUN apk add --no-cache ca-certificates tzdata || \
    (sleep 5 && apk add --no-cache ca-certificates tzdata) || \
    (sleep 10 && apk add --no-cache ca-certificates tzdata)

ENV TZ=Asia/Jakarta

WORKDIR /app

COPY --from=go-builder /build/ota-server .
COPY --from=go-builder /build/static ./static
COPY --from=svelte-builder /ui/build ./web

RUN mkdir -p /app/static/uploads

EXPOSE 9999

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9999/ || exit 1

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

ENTRYPOINT ["/app/ota-server"]