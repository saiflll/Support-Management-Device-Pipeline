# ============================================
# Forming Service Dockerfile
# Build context: servfor/ root (not forming/)
# Usage: docker build -f forming/Dockerfile .
# ============================================

# ============================================
# Stage 1: Build Svelte UI
# ============================================
FROM node:22-alpine AS svelte-builder
WORKDIR /ui

COPY core/forming/ui/package*.json ./
RUN npm ci

COPY core/forming/ui/ .

RUN npm run build

# ============================================
# Stage 2: Build Go Binary
# ============================================
FROM golang:1.24-alpine AS go-builder

WORKDIR /build

# Install dependencies (with retry for temporary network errors)
RUN apk add --no-cache git ca-certificates tzdata || \
    (sleep 5 && apk add --no-cache git ca-certificates tzdata) || \
    (sleep 10 && apk add --no-cache git ca-certificates tzdata)

COPY core/forming/go.mod core/forming/go.sum ./
RUN go mod download

COPY core/forming/ .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o forming-server .

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

COPY --from=go-builder /build/forming-server .
# Vite outDir is 'dist' (vite.config.js: build.outDir = '../web' relative to /ui → /web)
# But since Dockerfile copies forming/ui/ into /ui, outDir ../web = /web
COPY --from=svelte-builder /web ./web

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000/ || exit 1

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

CMD ["/app/forming-server"]
