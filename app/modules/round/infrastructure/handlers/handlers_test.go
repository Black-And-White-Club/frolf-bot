package roundhandlers

import (
	"testing"

	mockHelpers "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewRoundHandlers(t *testing.T) {
	// Define test cases
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
				mockRoundService := roundservice.NewMockService(ctrl)
				logger := loggerfrolfbot.NoOpLogger
				tracer := noop.NewTracerProvider().Tracer("test")
				mockHelpersInstance := mockHelpers.NewMockHelpers(ctrl)
				mockMetrics := mocks.NewMockRoundMetrics(ctrl)

				// Call the function being tested
				handlers := NewRoundHandlers(mockRoundService, logger, tracer, mockHelpersInstance, mockMetrics)

				// Ensure handlers are correctly created
				if handlers == nil {
					t.Fatalf("NewRoundHandlers returned nil")
				}

				// Access roundHandlers directly from the RoundHandlers struct
				roundHandlers := handlers.(*RoundHandlers)

				// Check that all dependencies were correctly assigned
				if roundHandlers.roundService != mockRoundService {
					t.Errorf("roundService not correctly assigned")
				}
				if roundHandlers.logger != logger {
					t.Errorf("logger not correctly assigned")
				}
				if roundHandlers.tracer != tracer {
					t.Errorf("tracer not correctly assigned")
				}
				if roundHandlers.helpers != mockHelpersInstance {
					t.Errorf("helpers not correctly assigned")
				}
				if roundHandlers.metrics != mockMetrics {
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
				handlers := NewRoundHandlers(nil, nil, nil, nil, nil)

				// Ensure handlers are correctly created
				if handlers == nil {
					t.Fatalf("NewRoundHandlers returned nil")
				}

				// Check nil fields
				if roundHandlers, ok := handlers.(*RoundHandlers); ok {
					if roundHandlers.roundService != nil {
						t.Errorf("roundService should be nil")
					}
					if roundHandlers.logger != nil {
						t.Errorf("logger should be nil")
					}
					if roundHandlers.tracer != nil {
						t.Errorf("tracer should be nil")
					}
					if roundHandlers.helpers != nil {
						t.Errorf("helpers should be nil")
					}
					if roundHandlers.metrics != nil {
						t.Errorf("metrics should be nil")
					}
				} else {
					t.Errorf("handlers is not of type *RoundHandlers")
				}
			},
		},
	}

	// Run all test cases
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
