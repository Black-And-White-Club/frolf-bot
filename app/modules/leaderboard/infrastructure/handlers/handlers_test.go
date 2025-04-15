package leaderboardhandlers

import (
	"context"
	"errors"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	mockHelpers "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
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

				mockLeaderboardService := leaderboardservice.NewMockService(ctrl)
				mockLogger := mocks.NewMockLogger(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)
				mockMetrics := mocks.NewMockLeaderboardMetrics(ctrl)

				handlers := NewLeaderboardHandlers(mockLeaderboardService, mockLogger, mockTracer, mockHelpers, mockMetrics)

				if handlers == nil {
					t.Fatalf("NewLeaderboardHandlers returned nil")
				}

				leaderboardHandlers := handlers.(*LeaderboardHandlers)

				if leaderboardHandlers.leaderboardService != mockLeaderboardService {
					t.Errorf("leaderboardService not correctly assigned")
				}
				if leaderboardHandlers.logger != mockLogger {
					t.Errorf("logger not correctly assigned")
				}
				if leaderboardHandlers.tracer != mockTracer {
					t.Errorf("tracer not correctly assigned")
				}
				if leaderboardHandlers.helpers != mockHelpers {
					t.Errorf("helpers not correctly assigned")
				}
				if leaderboardHandlers.metrics != mockMetrics {
					t.Errorf("metrics not correctly assigned")
				}

				if leaderboardHandlers.handlerWrapper == nil {
					t.Errorf("handlerWrapper should not be nil")
				}
			},
		},
		{
			name: "Handles nil dependencies",
			test: func(t *testing.T) {
				handlers := NewLeaderboardHandlers(nil, nil, nil, nil, nil)

				if handlers == nil {
					t.Fatalf("NewLeaderboardHandlers returned nil")
				}

				leaderboardHandlers := handlers.(*LeaderboardHandlers)

				if leaderboardHandlers.leaderboardService != nil {
					t.Errorf("leaderboardService should be nil")
				}
				if leaderboardHandlers.logger != nil {
					t.Errorf("logger should be nil")
				}
				if leaderboardHandlers.tracer != nil {
					t.Errorf("tracer should be nil")
				}
				if leaderboardHandlers.helpers != nil {
					t.Errorf("helpers should be nil")
				}
				if leaderboardHandlers.metrics != nil {
					t.Errorf("metrics should be nil")
				}

				if leaderboardHandlers.handlerWrapper == nil {
					t.Errorf("handlerWrapper should not be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func Test_handlerWrapper(t *testing.T) {
	type args struct {
		handlerName string
		unmarshalTo interface{}
		handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)
		logger      lokifrolfbot.Logger
		metrics     leaderboardmetrics.LeaderboardMetrics
		tracer      tempofrolfbot.Tracer
		helpers     utils.Helpers
	}
	tests := []struct {
		name    string
		args    func(ctrl *gomock.Controller) args
		wantErr bool
		setup   func(a *args) // Setup expectations per test
	}{
		{
			name: "Successful handler execution",
			args: func(ctrl *gomock.Controller) args {
				mockLogger := mocks.NewMockLogger(ctrl)
				mockMetrics := mocks.NewMockLeaderboardMetrics(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)

				return args{
					handlerName: "testHandler",
					unmarshalTo: nil,
					handlerFunc: func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
						return []*message.Message{msg}, nil
					},
					logger:  mockLogger,
					metrics: mockMetrics,
					tracer:  mockTracer,
					helpers: mockHelpers,
				}
			},
			wantErr: false,
			setup: func(a *args) {
				mockTracer := a.tracer.(*mocks.MockTracer)
				mockMetrics := a.metrics.(*mocks.MockLeaderboardMetrics)
				mockLogger := a.logger.(*mocks.MockLogger)

				mockTracer.EXPECT().StartSpan(
					gomock.AssignableToTypeOf(context.Background()),
					"testHandler",
					gomock.Any(),
				).Return(context.Background(), noop.Span{})

				mockMetrics.EXPECT().RecordHandlerAttempt("testHandler")
				mockLogger.EXPECT().Info("testHandler triggered", gomock.Any(), gomock.Any())
				mockMetrics.EXPECT().RecordHandlerDuration("testHandler", gomock.Any())
				mockLogger.EXPECT().Info("testHandler completed successfully", gomock.Any())
				mockMetrics.EXPECT().RecordHandlerSuccess("testHandler")
			},
		},
		{
			name: "Handler returns error",
			args: func(ctrl *gomock.Controller) args {
				mockLogger := mocks.NewMockLogger(ctrl)
				mockMetrics := mocks.NewMockLeaderboardMetrics(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)

				return args{
					handlerName: "testHandler",
					unmarshalTo: nil,
					handlerFunc: func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
						return nil, errors.New("handler error")
					},
					logger:  mockLogger,
					metrics: mockMetrics,
					tracer:  mockTracer,
					helpers: mockHelpers,
				}
			},
			wantErr: true,
			setup: func(a *args) {
				mockTracer := a.tracer.(*mocks.MockTracer)
				mockMetrics := a.metrics.(*mocks.MockLeaderboardMetrics)
				mockLogger := a.logger.(*mocks.MockLogger)

				mockTracer.EXPECT().StartSpan(
					gomock.AssignableToTypeOf(context.Background()),
					"testHandler",
					gomock.Any(),
				).Return(context.Background(), noop.Span{})

				mockMetrics.EXPECT().RecordHandlerAttempt("testHandler")
				mockLogger.EXPECT().Info("testHandler triggered", gomock.Any(), gomock.Any())
				mockMetrics.EXPECT().RecordHandlerDuration("testHandler", gomock.Any())
				mockLogger.EXPECT().Error("Error in testHandler", gomock.Any(), gomock.Any())
				mockMetrics.EXPECT().RecordHandlerFailure("testHandler")
			},
		},
		{
			name: "Unmarshal payload fails",
			args: func(ctrl *gomock.Controller) args {
				mockLogger := mocks.NewMockLogger(ctrl)
				mockMetrics := mocks.NewMockLeaderboardMetrics(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)

				return args{
					handlerName: "testHandler",
					unmarshalTo: &leaderboardevents.TagNumberRequestPayload{},
					handlerFunc: func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
						return []*message.Message{msg}, nil
					},
					logger:  mockLogger,
					metrics: mockMetrics,
					tracer:  mockTracer,
					helpers: mockHelpers,
				}
			},
			wantErr: true,
			setup: func(a *args) {
				mockTracer := a.tracer.(*mocks.MockTracer)
				mockMetrics := a.metrics.(*mocks.MockLeaderboardMetrics)
				mockLogger := a.logger.(*mocks.MockLogger)
				mockHelpers := a.helpers.(*mockHelpers.MockHelpers)

				mockTracer.EXPECT().StartSpan(
					gomock.AssignableToTypeOf(context.Background()),
					"testHandler",
					gomock.Any(),
				).Return(context.Background(), noop.Span{})

				mockMetrics.EXPECT().RecordHandlerAttempt("testHandler")
				mockLogger.EXPECT().Info("testHandler triggered", gomock.Any(), gomock.Any())
				mockMetrics.EXPECT().RecordHandlerDuration("testHandler", gomock.Any())
				mockLogger.EXPECT().Error("Failed to unmarshal payload", gomock.Any(), gomock.Any())
				mockMetrics.EXPECT().RecordHandlerFailure("testHandler")

				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal error"))
			},
		},
		{
			name: "Unmarshal payload succeeds",
			args: func(ctrl *gomock.Controller) args {
				mockLogger := mocks.NewMockLogger(ctrl)
				mockMetrics := mocks.NewMockLeaderboardMetrics(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)

				return args{
					handlerName: "testHandler",
					unmarshalTo: &leaderboardevents.TagNumberRequestPayload{},
					handlerFunc: func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
						return []*message.Message{msg}, nil
					},
					logger:  mockLogger,
					metrics: mockMetrics,
					tracer:  mockTracer,
					helpers: mockHelpers,
				}
			},
			wantErr: false,
			setup: func(a *args) {
				mockTracer := a.tracer.(*mocks.MockTracer)
				mockMetrics := a.metrics.(*mocks.MockLeaderboardMetrics)
				mockLogger := a.logger.(*mocks.MockLogger)
				mockHelpers := a.helpers.(*mockHelpers.MockHelpers)

				mockTracer.EXPECT().StartSpan(
					gomock.AssignableToTypeOf(context.Background()),
					"testHandler",
					gomock.Any(),
				).Return(context.Background(), noop.Span{})

				mockMetrics.EXPECT().RecordHandlerAttempt("testHandler")
				mockLogger.EXPECT().Info("testHandler triggered", gomock.Any(), gomock.Any())
				mockMetrics.EXPECT().RecordHandlerDuration("testHandler", gomock.Any())
				mockLogger.EXPECT().Info("testHandler completed successfully", gomock.Any())
				mockMetrics.EXPECT().RecordHandlerSuccess("testHandler")

				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			testArgs := tt.args(ctrl)

			if tt.setup != nil {
				tt.setup(&testArgs)
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
