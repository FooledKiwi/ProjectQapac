# ─── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Cache dependencies separately from source code
COPY go.mod go.sum ./
RUN go mod download

# Copy source and compile a statically-linked binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -extldflags=-static" \
    -o /out/server ./cmd/server/main.go

# ─── Runtime stage ────────────────────────────────────────────────────────────
FROM scratch

# Pull in TLS certificates and timezone data from the builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /out/server /server

EXPOSE 8080

# Required env vars (must be supplied at runtime):
#   DB_DSN       — PostgreSQL connection string
#   GOOGLE_API_KEY — Google Maps API key (optional, routing fallback)
#   PORT         — HTTP port (defaults to 8080 if unset)

ENTRYPOINT ["/server"]
