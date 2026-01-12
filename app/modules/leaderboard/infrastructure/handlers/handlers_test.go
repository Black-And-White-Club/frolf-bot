package leaderboardhandlers

import (
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewLeaderboardHandlers(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Creates handlers with all dependencies",
			test: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				// Create mock dependencies
				mockLeaderboardService := leaderboardservice.NewMockService(ctrl)
				logger := loggerfrolfbot.NoOpLogger
				tracer := noop.NewTracerProvider().Tracer("test")
				mockMetrics := &leaderboardmetrics.NoOpMetrics{}

				// Call the function being tested
				handlers := NewLeaderboardHandlers(mockLeaderboardService, logger, tracer, nil, mockMetrics)

				// Ensure handlers are correctly created
				if handlers == nil {
					t.Fatalf("NewLeaderboardHandlers returned nil")
				}

				// Access leaderboardHandlers directly from the LeaderboardHandlers struct
				leaderboardHandlers := handlers.(*LeaderboardHandlers)

				// Check that all dependencies were correctly assigned
				if leaderboardHandlers.leaderboardService != mockLeaderboardService {
					t.Errorf("leaderboardService not correctly assigned")
				}
				if leaderboardHandlers.logger != logger {
					t.Errorf("logger not correctly assigned")
				}
				if leaderboardHandlers.tracer != tracer {
					t.Errorf("tracer not correctly assigned")
				}
				if leaderboardHandlers.metrics != mockMetrics {
					t.Errorf("metrics not correctly assigned")
				}
			},
		},
		{
			name: "Handles nil dependencies",
			test: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				// Call with nil dependencies
				handlers := NewLeaderboardHandlers(nil, nil, nil, nil, nil)

				// Ensure handlers are correctly created
				if handlers == nil {
					t.Fatalf("NewLeaderboardHandlers returned nil")
				}

				// Check nil fields
				if leaderboardHandlers, ok := handlers.(*LeaderboardHandlers); ok {
					if leaderboardHandlers.leaderboardService != nil {
						t.Errorf("leaderboardService should be nil")
					}
					if leaderboardHandlers.logger != nil {
						t.Errorf("logger should be nil")
					}
					if leaderboardHandlers.tracer != nil {
						t.Errorf("tracer should be nil")
					}
					if leaderboardHandlers.metrics != nil {
						t.Errorf("metrics should be nil")
					}
				} else {
					t.Errorf("handlers is not of type *LeaderboardHandlers")
				}
			},
		},
	}

	// Run all test cases
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
