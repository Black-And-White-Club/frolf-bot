# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies and security updates
RUN apk add --no-cache --update git ca-certificates tzdata \
    && apk upgrade \
    && rm -rf /var/cache/apk/*

WORKDIR /app

# Copy go mod and sum files first for better caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application with optimizations and reproducible builds
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags='-w -s -extldflags "-static" -buildid=' \
    -trimpath \
    -a -installsuffix cgo \
    -o frolf-bot .

# Runtime stage - use distroless for minimal attack surface
FROM gcr.io/distroless/static:nonroot

# Add metadata labels for better container management
LABEL org.opencontainers.image.title="Frolf Bot" \
      org.opencontainers.image.description="Backend API for Disc Golf event management" \
      org.opencontainers.image.source="https://github.com/Black-And-White-Club/frolf-bot" \
      org.opencontainers.image.vendor="Black And White Club" \
      org.opencontainers.image.licenses="MIT"

# Copy timezone data and ca-certificates from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary from builder stage
COPY --from=builder /app/frolf-bot /frolf-bot

# Copy migrations for database setup
COPY --from=builder /app/app/modules/user/infrastructure/repositories/migrations /migrations/user
COPY --from=builder /app/app/modules/leaderboard/infrastructure/repositories/migrations /migrations/leaderboard
COPY --from=builder /app/app/modules/round/infrastructure/repositories/migrations /migrations/round
COPY --from=builder /app/app/modules/score/infrastructure/repositories/migrations /migrations/score

# Don't copy config.yaml - use environment variables or volume mounts instead
# COPY --from=builder /app/config.yaml /config.yaml

# Use nonroot user from distroless (UID 65532)
USER nonroot:nonroot

# No ports exposed - this is an event-driven service that communicates via NATS
# Health checks should be handled via your infrastructure/k8s setup

# Use exec form for better signal handling
ENTRYPOINT ["/frolf-bot"]
