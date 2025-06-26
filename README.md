# Frolf Bot

Backend service for Discord disc golf event management with event-driven architecture.

## Architecture

- **Event-driven microservice** using NATS for messaging
- **Modular design**: User, Round, Score, and Leaderboard modules
- **PostgreSQL** for data persistence with River for background jobs
- **OpenTelemetry** for observability (metrics, traces, logs)

## Configuration

The application supports both file-based configuration (`config.yaml`) and environment variables. For production/k8s deployments, use environment variables:

### Required Environment Variables

```bash
DATABASE_URL=postgres://user:password@host:port/database?sslmode=disable
NATS_URL=nats://host:port
METRICS_ADDRESS=0.0.0.0:4317
TEMPO_ENDPOINT=tempo:4317
ENV=production
```

### Optional Environment Variables

```bash
LOKI_URL=http://loki:3100
LOKI_TENANT_ID=1
TEMPO_INSECURE=true
TEMPO_SAMPLE_RATE=0.1
```

## Development

### Local Development Setup

1. **Install dependencies**
   ```bash
   go mod download
   ```

2. **Install River CLI** (for job queue migrations)
   ```bash
   go install github.com/riverqueue/river/cmd/river@latest
   ```

3. **Run migrations** (after your infrastructure is up)
   ```bash
   make migrate-all
   ```

4. **Run the application**
   ```bash
   go run main.go
   # or
   make run
   ```

### Testing

```bash
# All tests
make test-all-project

# Unit tests only
make test-unit-all

# Integration tests only
make test-integration-all
```

## Container

### Build

```bash
docker build -t frolf-bot .
```

### Run

```bash
docker run -d \
  --name frolf-bot \
  -e DATABASE_URL="postgres://user:pass@db:5432/frolf?sslmode=disable" \
  -e NATS_URL="nats://nats:4222" \
  -e METRICS_ADDRESS="0.0.0.0:4317" \
  -e TEMPO_ENDPOINT="tempo:4317" \
  -e ENV="production" \
  frolf-bot
```

## Key Features

- **Stateless and horizontally scalable**
- **Event-driven communication via NATS**
- **Built-in observability with OpenTelemetry**
- **Database migrations with River job queue support**
- **Graceful shutdown handling**

## Database Operations

```bash
# Run all migrations (River + application)
make migrate-all

# Rollback migrations
make rollback-all

# Clean everything
make clean-all
```

## Notes

- This is a **pure event-driven service** - no HTTP endpoints
- All observability is exported via OpenTelemetry
- Database setup and infrastructure dependencies are handled separately
- Designed for Kubernetes deployment
