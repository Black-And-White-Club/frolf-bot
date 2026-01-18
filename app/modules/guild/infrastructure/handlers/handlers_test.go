package guildhandlers

import (
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	guildmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewGuildHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 1. Setup Mock Service
	// If your GuildHandlers struct now uses the concrete *guildservice.GuildService,
	// you might need to cast the mock or ensure the interface satisfies the dependency.
	mockService := guildmocks.NewMockService(ctrl)

	// 2. Setup No-Op Observability
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &guildmetrics.NoOpMetrics{}

	// 3. Initialize Handlers
	// We pass nil for helpers if the test doesn't exercise them yet,
	// but usually it's better to pass a No-Op helper if available.
	handlers := NewGuildHandlers(mockService, logger, tracer, nil, metrics)

	// 4. Assertions
	if handlers == nil {
		t.Fatal("NewGuildHandlers returned nil")
	}

	// Check internal state
	if handlers.service != mockService {
		t.Errorf("expected service %v, got %v", mockService, handlers.service)
	}
}
