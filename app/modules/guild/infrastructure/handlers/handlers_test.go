package guildhandlers

import (
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewGuildHandlers(t *testing.T) {
	// 1. Setup Fake Service (instead of mock)
	fakeService := NewFakeGuildService()

	// 2. Setup No-Op Observability
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &guildmetrics.NoOpMetrics{}

	// 3. Initialize Handlers
	// Note: NewGuildHandlers returns the Handlers interface.
	// We cast to *GuildHandlers to check the internal state.
	handlersInterface := NewGuildHandlers(fakeService, logger, tracer, nil, metrics)
	handlers, ok := handlersInterface.(*GuildHandlers)

	// 4. Assertions
	if !ok {
		t.Fatal("NewGuildHandlers did not return a *GuildHandlers instance")
	}

	if handlers == nil {
		t.Fatal("NewGuildHandlers returned nil")
	}

	// Check internal state
	if handlers.service != fakeService {
		t.Errorf("expected service %v, got %v", fakeService, handlers.service)
	}
}
