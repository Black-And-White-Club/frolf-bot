package userhandlers

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	utilmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewUserHandlers(t *testing.T) {
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
				mockUserService := userservice.NewMockService(ctrl)
				mockLogger := loggerfrolfbot.NoOpLogger
				mockTracer := noop.NewTracerProvider().Tracer("test")
				mockHelpers := utilmocks.NewMockHelpers(ctrl)
				mockMetrics := mocks.NewMockUserMetrics(ctrl)

				// Call the function being tested
				handlers := NewUserHandlers(mockUserService, mockLogger, mockTracer, mockHelpers, mockMetrics)

				// Ensure handlers are correctly created
				if handlers == nil {
					t.Fatalf("NewUserHandlers returned nil")
				}

				// Access userHandlers directly from the UserHandlers struct
				userHandlers := handlers.(*UserHandlers)

				// Override handlerWrapper to prevent unwanted tracing/logging/metrics calls
				userHandlers.handlerWrapper = func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return func(msg *message.Message) ([]*message.Message, error) {
						// Directly call the handler function without any additional logic
						return handlerFunc(context.Background(), msg, unmarshalTo)
					}
				}

				// Check that all dependencies were correctly assigned
				if userHandlers.userService != mockUserService {
					t.Errorf("userService not correctly assigned")
				}
				if userHandlers.logger != mockLogger {
					t.Errorf("logger not correctly assigned")
				}
				if userHandlers.tracer != mockTracer {
					t.Errorf("tracer not correctly assigned")
				}
				if userHandlers.helpers != mockHelpers {
					t.Errorf("helpers not correctly assigned")
				}
				if userHandlers.metrics != mockMetrics {
					t.Errorf("metrics not correctly assigned")
				}

				// Ensure handlerWrapper is correctly set
				if userHandlers.handlerWrapper == nil {
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
				handlers := NewUserHandlers(nil, nil, nil, nil, nil)

				// Ensure handlers are correctly created
				if handlers == nil {
					t.Fatalf("NewUserHandlers returned nil")
				}

				// Check nil fields
				if userHandlers, ok := handlers.(*UserHandlers); ok {
					if userHandlers.userService != nil {
						t.Errorf("userService should be nil")
					}
					if userHandlers.logger != nil {
						t.Errorf("logger should be nil")
					}
					if userHandlers.tracer != nil {
						t.Errorf("tracer should be nil")
					}
					if userHandlers.helpers != nil {
						t.Errorf("helpers should be nil")
					}
					if userHandlers.metrics != nil {
						t.Errorf("metrics should be nil")
					}

					// Ensure handlerWrapper is still set
					if userHandlers.handlerWrapper == nil {
						t.Errorf("handlerWrapper should not be nil")
					}
				} else {
					t.Errorf("handlers is not of type * User  Handlers")
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
		metrics     usermetrics.UserMetrics
		tracer      trace.Tracer
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
				mockLogger := loggerfrolfbot.NoOpLogger
				mockMetrics := mocks.NewMockUserMetrics(ctrl)
				mockTracer := noop.NewTracerProvider().Tracer("test")
				mockHelpers := utilmocks.NewMockHelpers(ctrl)

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
				mockMetrics := a.metrics.(*mocks.MockUserMetrics)

				//  Use gomock.Any() to match any context.
				mockMetrics.EXPECT().RecordHandlerAttempt(gomock.Any(), "testHandler")
				mockMetrics.EXPECT().RecordHandlerDuration(gomock.Any(), "testHandler", gomock.Any())
				mockMetrics.EXPECT().RecordHandlerSuccess(gomock.Any(), "testHandler")
			},
		},
		{
			name: "Handler returns error",
			args: func(ctrl *gomock.Controller) args {
				mockLogger := loggerfrolfbot.NoOpLogger
				mockMetrics := mocks.NewMockUserMetrics(ctrl)
				mockTracer := noop.NewTracerProvider().Tracer("test")
				mockHelpers := utilmocks.NewMockHelpers(ctrl)

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
				mockMetrics := a.metrics.(*mocks.MockUserMetrics)

				//  Use gomock.Any() to match any context.
				mockMetrics.EXPECT().RecordHandlerAttempt(gomock.Any(), "testHandler")
				mockMetrics.EXPECT().RecordHandlerDuration(gomock.Any(), "testHandler", gomock.Any())
				mockMetrics.EXPECT().RecordHandlerFailure(gomock.Any(), "testHandler")
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
