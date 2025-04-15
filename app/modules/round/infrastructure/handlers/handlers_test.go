package roundhandlers

import (
	"context"
	"errors"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	mockHelpers "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewRoundHandlers(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Creates handlers with all dependencies",
			test: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockRoundService := roundservice.NewMockService(ctrl)
				mockLogger := mocks.NewMockLogger(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)
				mockMetrics := mocks.NewMockRoundMetrics(ctrl)

				handlers := NewRoundHandlers(mockRoundService, mockLogger, mockTracer, mockHelpers, mockMetrics)

				if handlers == nil {
					t.Fatalf("NewRoundHandlers returned nil")
				}

				roundHandlers := handlers.(*RoundHandlers)

				if roundHandlers.roundService != mockRoundService {
					t.Errorf("roundService not correctly assigned")
				}
				if roundHandlers.logger != mockLogger {
					t.Errorf("logger not correctly assigned")
				}
				if roundHandlers.tracer != mockTracer {
					t.Errorf("tracer not correctly assigned")
				}
				if roundHandlers.helpers != mockHelpers {
					t.Errorf("helpers not correctly assigned")
				}
				if roundHandlers.metrics != mockMetrics {
					t.Errorf("metrics not correctly assigned")
				}

				if roundHandlers.handlerWrapper == nil {
					t.Errorf("handlerWrapper should not be nil")
				}
			},
		},
		{
			name: "Handles nil dependencies",
			test: func(t *testing.T) {
				handlers := NewRoundHandlers(nil, nil, nil, nil, nil)

				if handlers == nil {
					t.Fatalf("NewRoundHandlers returned nil")
				}

				roundHandlers := handlers.(*RoundHandlers)

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

				if roundHandlers.handlerWrapper == nil {
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
		metrics     roundmetrics.RoundMetrics
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
				mockMetrics := mocks.NewMockRoundMetrics(ctrl)
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
				mockMetrics := a.metrics.(*mocks.MockRoundMetrics)
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
				mockMetrics := mocks.NewMockRoundMetrics(ctrl)
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
				mockMetrics := a.metrics.(*mocks.MockRoundMetrics)
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
				mockMetrics := mocks.NewMockRoundMetrics(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)

				return args{
					handlerName: "testHandler",
					unmarshalTo: &roundevents.RoundCreatedPayload{},
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
				mockMetrics := a.metrics.(*mocks.MockRoundMetrics)
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
				mockMetrics := mocks.NewMockRoundMetrics(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)

				return args{
					handlerName: "testHandler",
					unmarshalTo: &roundevents.RoundCreatedPayload{},
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
				mockMetrics := a.metrics.(*mocks.MockRoundMetrics)
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
