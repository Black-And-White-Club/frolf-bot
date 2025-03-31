package scorehandlers

import (
	"context"
	"errors"
	"testing"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	mockHelpers "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/score"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewScoreHandlers(t *testing.T) {
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
				mockScoreService := scoreservice.NewMockService(ctrl)
				mockLogger := mocks.NewMockLogger(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)
				mockMetrics := mocks.NewMockScoreMetrics(ctrl)

				// Call the function being tested
				handlers := NewScoreHandlers(mockScoreService, mockLogger, mockTracer, mockHelpers, mockMetrics)

				// Ensure handlers are correctly created
				if handlers == nil {
					t.Fatalf("NewScoreHandlers returned nil")
				}

				// Access scoreHandlers directly from the ScoreHandlers struct
				scoreHandlers := handlers.(*ScoreHandlers)

				// Override handlerWrapper to prevent unwanted tracing/logging/metrics calls
				scoreHandlers.handlerWrapper = func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return func(msg *message.Message) ([]*message.Message, error) {
						// Directly call the handler function without any additional logic
						return handlerFunc(context.Background(), msg, unmarshalTo)
					}
				}

				// Check that all dependencies were correctly assigned
				if scoreHandlers.scoreService != mockScoreService {
					t.Errorf("scoreService not correctly assigned")
				}
				if scoreHandlers.logger != mockLogger {
					t.Errorf("logger not correctly assigned")
				}
				if scoreHandlers.tracer != mockTracer {
					t.Errorf("tracer not correctly assigned")
				}
				if scoreHandlers.helpers != mockHelpers {
					t.Errorf("helpers not correctly assigned")
				}
				if scoreHandlers.metrics != mockMetrics {
					t.Errorf("metrics not correctly assigned")
				}

				// Ensure handlerWrapper is correctly set
				if scoreHandlers.handlerWrapper == nil {
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
				handlers := NewScoreHandlers(nil, nil, nil, nil, nil)

				// Ensure handlers are correctly created
				if handlers == nil {
					t.Fatalf("NewScoreHandlers returned nil")
				}

				// Check nil fields
				if scoreHandlers, ok := handlers.(*ScoreHandlers); ok {
					if scoreHandlers.scoreService != nil {
						t.Errorf("scoreService should be nil")
					}
					if scoreHandlers.logger != nil {
						t.Errorf("logger should be nil")
					}
					if scoreHandlers.tracer != nil {
						t.Errorf("tracer should be nil")
					}
					if scoreHandlers.helpers != nil {
						t.Errorf("helpers should be nil")
					}
					if scoreHandlers.metrics != nil {
						t.Errorf("metrics should be nil")
					}

					// Ensure handlerWrapper is still set
					if scoreHandlers.handlerWrapper == nil {
						t.Errorf("handlerWrapper should not be nil")
					}
				} else {
					t.Errorf("handlers is not of type *ScoreHandlers")
				}
			},
		},
	}

	// Run all test cases
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestHandlerWrapper(t *testing.T) {
	type args struct {
		handlerName string
		unmarshalTo interface{}
		handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)
		logger      lokifrolfbot.Logger
		metrics     scoremetrics.ScoreMetrics
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
				mockMetrics := mocks.NewMockScoreMetrics(ctrl)
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
			wantErr: true,
			setup: func(a *args) {
				mockTracer := a.tracer.(*mocks.MockTracer)
				mockMetrics := a.metrics.(*mocks.MockScoreMetrics)
				mockLogger := a.logger.(*mocks.MockLogger)

				// Mock tracer.StartSpan
				mockTracer.EXPECT().StartSpan(
					gomock.AssignableToTypeOf(context.Background()),
					"testHandler",
					gomock.Any(),
				).Return(context.Background(), noop.Span{})

				// Mock metrics & logs
				mockMetrics.EXPECT().RecordHandlerAttempt("testHandler")
				mockLogger.EXPECT().Info("testHandler triggered", gomock.Any(), gomock.Any())
				mockMetrics.EXPECT().RecordHandlerDuration("testHandler", gomock.Any())
				mockLogger.EXPECT().Error("No payload instance provided", gomock.Any())
				mockMetrics.EXPECT().RecordHandlerFailure("testHandler")
			},
		},
		{
			name: "Handler returns error",
			args: func(ctrl *gomock.Controller) args {
				mockLogger := mocks.NewMockLogger(ctrl)
				mockMetrics := mocks.NewMockScoreMetrics(ctrl)
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
				mockMetrics := a.metrics.(*mocks.MockScoreMetrics)
				mockLogger := a.logger.(*mocks.MockLogger)

				// Mock tracer.StartSpan
				mockTracer.EXPECT().StartSpan(
					gomock.AssignableToTypeOf(context.Background()),
					"testHandler",
					gomock.Any(),
				).Return(context.Background(), noop.Span{})

				// Mock metrics & logs
				mockMetrics.EXPECT().RecordHandlerAttempt("testHandler")
				mockLogger.EXPECT().Info("testHandler triggered", gomock.Any(), gomock.Any())
				mockMetrics.EXPECT().RecordHandlerDuration("testHandler", gomock.Any())
				mockLogger.EXPECT().Error("No payload instance provided", gomock.Any())
				mockMetrics.EXPECT().RecordHandlerFailure("testHandler")
			},
		},
		{
			name: "Unmarshal payload fails",
			args: func(ctrl *gomock.Controller) args {
				mockLogger := mocks.NewMockLogger(ctrl)
				mockMetrics := mocks.NewMockScoreMetrics(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)

				return args{
					handlerName: "testHandler",
					unmarshalTo: &scoreevents.ScoreUpdateRequestPayload{},
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
				mockMetrics := a.metrics.(*mocks.MockScoreMetrics)
				mockLogger := a.logger.(*mocks.MockLogger)
				mockHelpers := a.helpers.(*mockHelpers.MockHelpers)

				// Mock tracer.StartSpan
				mockTracer.EXPECT().StartSpan(
					gomock.AssignableToTypeOf(context.Background()),
					"testHandler",
					gomock.Any(),
				).Return(context.Background(), noop.Span{})

				// Mock metrics & logs
				mockMetrics.EXPECT().RecordHandlerAttempt("testHandler")
				mockLogger.EXPECT().Info("testHandler triggered", gomock.Any(), gomock.Any())
				mockMetrics.EXPECT().RecordHandlerDuration("testHandler", gomock.Any())
				mockLogger.EXPECT().Error("Failed to unmarshal payload", gomock.Any(), gomock.Any())
				mockMetrics.EXPECT().RecordHandlerFailure("testHandler")

				// Mock unmarshal payload
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal error"))
			},
		},
		{
			name: "Unmarshal payload succeeds",
			args: func(ctrl *gomock.Controller) args {
				mockLogger := mocks.NewMockLogger(ctrl)
				mockMetrics := mocks.NewMockScoreMetrics(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)

				return args{
					handlerName: "testHandler",
					unmarshalTo: &scoreevents.ScoreUpdateRequestPayload{},
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
				mockMetrics := a.metrics.(*mocks.MockScoreMetrics)
				mockLogger := a.logger.(*mocks.MockLogger)
				mockHelpers := a.helpers.(*mockHelpers.MockHelpers)

				// Mock tracer.StartSpan
				mockTracer.EXPECT().StartSpan(
					gomock.AssignableToTypeOf(context.Background()),
					"testHandler",
					gomock.Any(),
				).Return(context.Background(), noop.Span{})

				// Mock metrics & logs
				mockMetrics.EXPECT().RecordHandlerAttempt("testHandler")
				mockLogger.EXPECT().Info("testHandler triggered", gomock.Any(), gomock.Any())
				mockMetrics.EXPECT().RecordHandlerDuration("testHandler", gomock.Any())
				mockLogger.EXPECT().Info("testHandler completed successfully", gomock.Any())
				mockMetrics.EXPECT().RecordHandlerSuccess("testHandler")

				// Mock unmarshal payload
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Initialize args using fresh mock controller
			testArgs := tt.args(ctrl)

			// Set up expectations
			if tt.setup != nil {
				tt.setup(&testArgs)
			}

			// Run handlerWrapper
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
