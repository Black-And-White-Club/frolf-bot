package scorehandlers

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	mockHelpers "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace"
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
				logger := loggerfrolfbot.NoOpLogger
				tracer := noop.NewTracerProvider().Tracer("test")
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)
				mockMetrics := mocks.NewMockScoreMetrics(ctrl)

				// Call the function being tested
				handlers := NewScoreHandlers(mockScoreService, logger, tracer, mockHelpers, mockMetrics)

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
				if scoreHandlers.logger != logger {
					t.Errorf("logger not correctly assigned")
				}
				if scoreHandlers.tracer != tracer {
					t.Errorf("tracer not correctly assigned")
				}
				if scoreHandlers.Helpers != mockHelpers {
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
					if scoreHandlers.Helpers != nil {
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
		logger      *slog.Logger
		metrics     scoremetrics.ScoreMetrics
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
				mockMetrics := mocks.NewMockScoreMetrics(ctrl)
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
				mockMetrics := a.metrics.(*mocks.MockScoreMetrics)

				// Mock metrics & logs
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
				mockMetrics := mocks.NewMockScoreMetrics(ctrl)
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
				mockMetrics := a.metrics.(*mocks.MockScoreMetrics)

				// Mock metrics & logs
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
				mockMetrics := mocks.NewMockScoreMetrics(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)

				return args{
					handlerName: "testHandler",
					unmarshalTo: &scoreevents.ScoreUpdateRequestPayload{},
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
				mockMetrics := a.metrics.(*mocks.MockScoreMetrics)
				mockHelpers := a.helpers.(*mockHelpers.MockHelpers)

				// Mock metrics & logs
				mockMetrics.EXPECT().RecordHandlerAttempt(gomock.Any(), "testHandler")
				mockMetrics.EXPECT().RecordHandlerDuration(gomock.Any(), "testHandler", gomock.Any())
				mockMetrics.EXPECT().RecordHandlerFailure(gomock.Any(), "testHandler")

				// Mock unmarshal payload
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal error"))
			},
		},
		{
			name: "Unmarshal payload succeeds",
			args: func(ctrl *gomock.Controller) args {
				logger := loggerfrolfbot.NoOpLogger
				tracer := noop.NewTracerProvider().Tracer("test")
				mockMetrics := mocks.NewMockScoreMetrics(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)

				return args{
					handlerName: "testHandler",
					unmarshalTo: &scoreevents.ScoreUpdateRequestPayload{},
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
				mockMetrics := a.metrics.(*mocks.MockScoreMetrics)
				mockHelpers := a.helpers.(*mockHelpers.MockHelpers)

				// Mock metrics & logs
				mockMetrics.EXPECT().RecordHandlerAttempt(gomock.Any(), "testHandler")
				mockMetrics.EXPECT().RecordHandlerDuration(gomock.Any(), "testHandler", gomock.Any())
				mockMetrics.EXPECT().RecordHandlerSuccess(gomock.Any(), "testHandler")

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

			// Create a context for the test
			ctx := context.Background()

			// Set up expectations
			if tt.setup != nil {
				tt.setup(&testArgs, ctx)
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
