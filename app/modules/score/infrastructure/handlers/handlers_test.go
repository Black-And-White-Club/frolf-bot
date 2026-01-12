package scorehandlers

import (
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewScoreHandlers(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Creates handlers with all dependencies",
			test: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockScoreService := scoreservice.NewMockService(ctrl)
				logger := loggerfrolfbot.NoOpLogger
				tracer := noop.NewTracerProvider().Tracer("test")
				mockMetrics := &scoremetrics.NoOpMetrics{}

				handlers := NewScoreHandlers(mockScoreService, logger, tracer, nil, mockMetrics)

				if handlers == nil {
					t.Fatalf("NewScoreHandlers returned nil")
				}

				scoreHandlers := handlers.(*ScoreHandlers)

				if scoreHandlers.scoreService != mockScoreService {
					t.Errorf("scoreService not correctly assigned")
				}
				if scoreHandlers.logger != logger {
					t.Errorf("logger not correctly assigned")
				}
				if scoreHandlers.tracer != tracer {
					t.Errorf("tracer not correctly assigned")
				}
				if scoreHandlers.metrics != mockMetrics {
					t.Errorf("metrics not correctly assigned")
				}
			},
		},
		{
			name: "Handles nil dependencies",
			test: func(t *testing.T) {
				handlers := NewScoreHandlers(nil, nil, nil, nil, nil)

				if handlers == nil {
					t.Fatalf("NewScoreHandlers returned nil")
				}

				scoreHandlers, ok := handlers.(*ScoreHandlers)
				if !ok {
					t.Fatalf("handlers is not of type *ScoreHandlers")
				}
				if scoreHandlers.scoreService != nil {
					t.Errorf("scoreService should be nil")
				}
				if scoreHandlers.logger != nil {
					t.Errorf("logger should be nil")
				}
				if scoreHandlers.tracer != nil {
					t.Errorf("tracer should be nil")
				}
				if scoreHandlers.Helpers != nil {
					t.Errorf("helpers should be nil")
				}
				if scoreHandlers.metrics != nil {
					t.Errorf("metrics should be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
