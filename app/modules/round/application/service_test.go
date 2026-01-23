package roundservice

import (
	"context"
	"log/slog"
	"testing"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	queuemocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/queue/mocks"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

type testUserLookup struct{}

func (testUserLookup) FindByNormalizedUDiscUsername(ctx context.Context, guildID sharedtypes.GuildID, normalizedUsername string) (*UserIdentity, error) {
	return nil, nil
}

func (testUserLookup) FindByNormalizedUDiscDisplayName(ctx context.Context, guildID sharedtypes.GuildID, normalizedDisplayName string) (*UserIdentity, error) {
	return nil, nil
}

func (testUserLookup) FindByPartialUDiscName(ctx context.Context, guildID sharedtypes.GuildID, partialName string) ([]*UserIdentity, error) {
	return nil, nil
}

func TestNewRoundService(t *testing.T) {
	// Define test cases
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Creates service with all dependencies",
			test: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				// Create mock dependencies
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				mockDB := rounddb.NewMockRepository(ctrl)
				mockQueueService := queuemocks.NewMockQueueService(ctrl)
				mockMetrics := &roundmetrics.NoOpMetrics{}
				tracer := noop.NewTracerProvider().Tracer("test")
				mockEventbus := mocks.NewMockEventBus(ctrl)
				mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)
				userLookup := testUserLookup{}

				// Call the function being tested
				service := NewRoundService(mockDB, mockQueueService, mockEventbus, userLookup, mockMetrics, logger, tracer, mockRoundValidator)

				// Ensure service is correctly created
				if service == nil {
					t.Fatalf("NewRoundService returned nil")
				}

				// Check that all dependencies were correctly assigned
				if service.repo != mockDB {
					t.Errorf("Round DB not correctly assigned")
				}
				if service.queueService != mockQueueService {
					t.Errorf("Queue Service not correctly assigned")
				}
				if service.logger != logger {
					t.Errorf("logger not correctly assigned")
				}
				if service.metrics != mockMetrics {
					t.Errorf("metrics not correctly assigned")
				}
				if service.tracer != tracer {
					t.Errorf("tracer not correctly assigned")
				}
				if service.userLookup != userLookup {
					t.Errorf("userLookup not correctly assigned")
				}
			},
		},
		{
			name: "Handles nil dependencies",
			test: func(t *testing.T) {
				// Call with nil dependencies
				service := NewRoundService(nil, nil, nil, nil, nil, nil, nil, nil)

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
