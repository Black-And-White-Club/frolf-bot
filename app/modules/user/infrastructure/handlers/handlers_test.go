package userhandlers

import (
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewUserHandlers(t *testing.T) {
	// Arrange dependencies using Fakes/NoOps
	fakeService := NewFakeUserService()
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &usermetrics.NoOpMetrics{}

	t.Run("Initialize with all dependencies", func(t *testing.T) {
		handlers := NewUserHandlers(fakeService, logger, tracer, nil, metrics)
		if handlers == nil {
			t.Fatal("Expected non-nil handlers")
		}

		// Verify the underlying type matches
		if _, ok := handlers.(*UserHandlers); !ok {
			t.Error("Expected handlers to be of type *UserHandlers")
		}
	})

	t.Run("Initialize with nil dependencies", func(t *testing.T) {
		// Even with nils, the factory should return an instance
		// (though calling methods on it might panic depending on implementation)
		handlersNil := NewUserHandlers(nil, nil, nil, nil, nil)
		if handlersNil == nil {
			t.Fatal("Expected non-nil handlers even with nil dependencies")
		}
	})
}
