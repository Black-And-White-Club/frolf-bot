package roundservice

import (
	"log/slog"
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewRoundService(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Creates service with all dependencies",
			test: func(t *testing.T) {
				// Create fake dependencies
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				fakeRepo := &FakeRepo{}
				fakeQueue := &FakeQueueService{}
				fakeEventBus := &FakeEventBus{}
				fakeMetrics := &roundmetrics.NoOpMetrics{}
				tracer := noop.NewTracerProvider().Tracer("test")
				fakeValidator := &FakeRoundValidator{}
				fakeUserLookup := &FakeUserLookup{}

				// Call the function being tested
				service := NewRoundService(fakeRepo, fakeQueue, fakeEventBus, fakeUserLookup, fakeMetrics, logger, tracer, fakeValidator, nil)

				// Ensure service is correctly created
				if service == nil {
					t.Fatalf("NewRoundService returned nil")
				}

				// Check that all dependencies were correctly assigned
				if service.repo != fakeRepo {
					t.Errorf("Round DB not correctly assigned")
				}
				if service.queueService != fakeQueue {
					t.Errorf("Queue Service not correctly assigned")
				}
				if service.logger != logger {
					t.Errorf("logger not correctly assigned")
				}
				if service.metrics != fakeMetrics {
					t.Errorf("metrics not correctly assigned")
				}
				if service.tracer != tracer {
					t.Errorf("tracer not correctly assigned")
				}
				if service.userLookup != fakeUserLookup {
					t.Errorf("userLookup not correctly assigned")
				}
			},
		},
		{
			name: "Handles nil dependencies",
			test: func(t *testing.T) {
				// Call with nil dependencies
				service := NewRoundService(nil, nil, nil, nil, nil, nil, nil, nil, nil)

				// Ensure service is correctly created
				if service == nil {
					t.Fatalf("NewRoundService returned nil")
				}

				// Check nil fields
				if service.repo != nil {
					t.Errorf("Round DB should be nil")
				}
				if service.queueService != nil {
					t.Errorf("Queue Service should be nil")
				}
				if service.eventBus != nil {
					t.Errorf("EventBus should be nil")
				}
				if service.logger != nil {
					t.Errorf("logger should be nil")
				}
				if service.metrics != nil {
					t.Errorf("metrics should be nil")
				}
				if service.tracer != nil {
					t.Errorf("tracer should be nil")
				}
			},
		},
	}

	// Run all test cases
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
