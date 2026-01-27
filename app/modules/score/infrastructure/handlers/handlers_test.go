package scorehandlers

import (
	"log/slog"
	"testing"

	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewScoreHandlers(t *testing.T) {
	// Arrange dependencies using Fakes and NoOps
	fakeService := NewFakeScoreService()
	logger := slog.Default() // Or use a NoOp logger if defined in your shared utils
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &scoremetrics.NoOpMetrics{}

	t.Run("Initialize with all dependencies", func(t *testing.T) {
		handlers := NewScoreHandlers(fakeService, logger, tracer, nil, metrics)
		if handlers == nil {
			t.Fatal("Expected non-nil handlers")
		}

		// Verify the underlying type matches and dependencies are assigned
		scoreHandlers, ok := handlers.(*ScoreHandlers)
		if !ok {
			t.Error("Expected handlers to be of type *ScoreHandlers")
		}

		if scoreHandlers.service != fakeService {
			t.Errorf("Expected service to be assigned to FakeScoreService")
		}
	})

	t.Run("Initialize with nil dependencies", func(t *testing.T) {
		// The factory should still return a struct instance even if fields are nil
		handlersNil := NewScoreHandlers(nil, nil, nil, nil, nil)
		if handlersNil == nil {
			t.Fatal("Expected non-nil handlers even with nil dependencies")
		}

		scoreHandlers, ok := handlersNil.(*ScoreHandlers)
		if !ok {
			t.Fatal("handlers is not of type *ScoreHandlers")
		}
		if scoreHandlers.service != nil {
			t.Errorf("service should be nil")
		}
	})
}
