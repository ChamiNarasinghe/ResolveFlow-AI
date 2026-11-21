# syntax=docker/dockerfile:1

# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

# git is required by `go mod download` for VCS-stamped modules.
RUN apk add --no-cache git

WORKDIR /build

# Cache dependency downloads separately from source compilation.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and compile a fully static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /app/server ./cmd/api

# ── Stage 2: runtime ──────────────────────────────────────────────────────────
FROM alpine:3.21 AS runtime

# ca-certificates: required for TLS connections to external services.
# tzdata: allows TZ env var to set the container timezone.
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Binary from the builder stage.
COPY --from=builder /app/server .

# Migration SQL files — kept in the image so the migrate CLI can be run
# as a one-off command against the same image without a separate volume.
COPY migrations/ ./migrations/

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["./server"]
