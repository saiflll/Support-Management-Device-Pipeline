FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bridge .

FROM alpine:3.19

# Install python and pip to run headroom-ai
# python3 -m venv sudah built-in di stdlib (tanpa paket py3-venv terpisah)
RUN apk add --no-cache python3 py3-pip curl && \
    python3 -m venv /opt/venv && \
    /opt/venv/bin/pip install --no-cache-dir "headroom-ai[proxy]"

# Tambah PATH untuk akses headroom CLI dari venv
ENV PATH="/opt/venv/bin:$PATH"

COPY --from=builder /bridge /bridge
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

EXPOSE 8787

WORKDIR /app
CMD ["/app/entrypoint.sh"]
