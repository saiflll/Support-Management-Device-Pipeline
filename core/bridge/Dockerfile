FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bridge .

FROM alpine:3.19

# Install python and pip to run headroom-ai
# Install global (tanpa virtualenv) — simpel untuk container
RUN apk add --no-cache python3 py3-pip curl && \
    pip3 install --no-cache-dir "headroom-ai[proxy]"

COPY --from=builder /bridge /bridge
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

EXPOSE 8787

WORKDIR /app
CMD ["/app/entrypoint.sh"]
