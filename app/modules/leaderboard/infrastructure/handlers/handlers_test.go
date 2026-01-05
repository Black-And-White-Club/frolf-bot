package leaderboardhandlers

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	mockHelpers "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace"
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
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)
				mockMetrics := mocks.NewMockLeaderboardMetrics(ctrl)

				// Call the function being tested
				handlers := NewLeaderboardHandlers(mockLeaderboardService, logger, tracer, mockHelpers, mockMetrics)

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
				if leaderboardHandlers.Helpers != mockHelpers {
					t.Errorf("helpers not correctly assigned")
				}
				if leaderboardHandlers.metrics != mockMetrics {
					t.Errorf("metrics not correctly assigned")
				}

				// Ensure handlerWrapper is correctly set
				if leaderboardHandlers.handlerWrapper == nil {
					t.Errorf("handlerWrapper should not be nil")
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
					if leaderboardHandlers.Helpers != nil {
						t.Errorf("helpers should be nil")
					}
					if leaderboardHandlers.metrics != nil {
						t.Errorf("metrics should be nil")
					}

					// Ensure handlerWrapper is still set
					if leaderboardHandlers.handlerWrapper == nil {
						t.Errorf("handlerWrapper should not be nil")
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

func Test_handlerWrapper(t *testing.T) {
	type args struct {
		handlerName string
		unmarshalTo interface{}
		handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)
		logger      *slog.Logger
		metrics     leaderboardmetrics.LeaderboardMetrics
		tracer      trace.Tracer
		helpers     utils.Helpers
	}

	tests := []struct {
		name    string
		args    func(ctrl *gomock.Controller) args
		wantErr bool
		setup   func(a *args, ctx context.Context) // Setup expectations per test
	}{
		{
			name: "Successful handler execution",
			args: func(ctrl *gomock.Controller) args {
				logger := loggerfrolfbot.NoOpLogger
				tracer := noop.NewTracerProvider().Tracer("test")
				mockMetrics := mocks.NewMockLeaderboardMetrics(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)

				return args{
					handlerName: "testHandler",
					unmarshalTo: nil,
					handlerFunc: func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
						return []*message.Message{msg}, nil
					},
					logger:  logger,
					metrics: mockMetrics,
					tracer:  tracer,
					helpers: mockHelpers,
				}
			},
			wantErr: false,
			setup: func(a *args, ctx context.Context) {
				mockMetrics := a.metrics.(*mocks.MockLeaderboardMetrics)

				mockMetrics.EXPECT().RecordHandlerAttempt(gomock.Any(), "testHandler")
				mockMetrics.EXPECT().RecordHandlerDuration(gomock.Any(), "testHandler", gomock.Any())
				mockMetrics.EXPECT().RecordHandlerSuccess(gomock.Any(), "testHandler")
			},
		},
		{
			name: "Handler returns error",
			args: func(ctrl *gomock.Controller) args {
				logger := loggerfrolfbot.NoOpLogger
				tracer := noop.NewTracerProvider().Tracer("test")
				mockMetrics := mocks.NewMockLeaderboardMetrics(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)

				return args{
					handlerName: "testHandler",
					unmarshalTo: nil,
					handlerFunc: func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
						return nil, errors.New("handler error")
					},
					logger:  logger,
					metrics: mockMetrics,
					tracer:  tracer,
					helpers: mockHelpers,
				}
			},
			wantErr: true,
			setup: func(a *args, ctx context.Context) {
				mockMetrics := a.metrics.(*mocks.MockLeaderboardMetrics)

				mockMetrics.EXPECT().RecordHandlerAttempt(gomock.Any(), "testHandler")
				mockMetrics.EXPECT().RecordHandlerDuration(gomock.Any(), "testHandler", gomock.Any())
				mockMetrics.EXPECT().RecordHandlerFailure(gomock.Any(), "testHandler")
			},
		},
		{
			name: "Unmarshal payload fails",
			args: func(ctrl *gomock.Controller) args {
				logger := loggerfrolfbot.NoOpLogger
				tracer := noop.NewTracerProvider().Tracer("test")
				mockMetrics := mocks.NewMockLeaderboardMetrics(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)

				return args{
					handlerName: "testHandler",
					unmarshalTo: &leaderboardevents.TagNumberRequestPayloadV1{},
					handlerFunc: func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
						return []*message.Message{msg}, nil
					},
					logger:  logger,
					metrics: mockMetrics,
					tracer:  tracer,
					helpers: mockHelpers,
				}
			},
			wantErr: true,
			setup: func(a *args, ctx context.Context) {
				mockMetrics := a.metrics.(*mocks.MockLeaderboardMetrics)
				mockHelpers := a.helpers.(*mockHelpers.MockHelpers)

				mockMetrics.EXPECT().RecordHandlerAttempt(gomock.Any(), "testHandler")
				mockMetrics.EXPECT().RecordHandlerDuration(gomock.Any(), "testHandler", gomock.Any())
				mockMetrics.EXPECT().RecordHandlerFailure(gomock.Any(), "testHandler")

				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal error"))
			},
		},
		{
			name: "Unmarshal payload succeeds",
			args: func(ctrl *gomock.Controller) args {
				logger := loggerfrolfbot.NoOpLogger
				tracer := noop.NewTracerProvider().Tracer("test")
				mockMetrics := mocks.NewMockLeaderboardMetrics(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)

				return args{
					handlerName: "testHandler",
					unmarshalTo: &leaderboardevents.TagNumberRequestPayloadV1{},
					handlerFunc: func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
						return []*message.Message{msg}, nil
					},
					logger:  logger,
					metrics: mockMetrics,
					tracer:  tracer,
					helpers: mockHelpers,
				}
			},
			wantErr: false,
			setup: func(a *args, ctx context.Context) {
				mockMetrics := a.metrics.(*mocks.MockLeaderboardMetrics)
				mockHelpers := a.helpers.(*mockHelpers.MockHelpers)

				mockMetrics.EXPECT().RecordHandlerAttempt(gomock.Any(), "testHandler")
				mockMetrics.EXPECT().RecordHandlerDuration(gomock.Any(), "testHandler", gomock.Any())
				mockMetrics.EXPECT().RecordHandlerSuccess(gomock.Any(), "testHandler")

				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			testArgs := tt.args(ctrl)
			ctx := context.Background()

			if tt.setup != nil {
				tt.setup(&testArgs, ctx)
			}

			handlerFunc := handlerWrapper(
				testArgs.handlerName,
				testArgs.unmarshalTo,
				testArgs.handlerFunc,
				testArgs.logger,
				testArgs.metrics,
				testArgs.tracer,
				testArgs.helpers,
			)

			msg := message.NewMessage("test-id", nil)
			_, err := handlerFunc(msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("handlerWrapper() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
