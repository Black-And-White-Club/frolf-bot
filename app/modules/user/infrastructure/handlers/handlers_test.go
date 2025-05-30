package userhandlers

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	mockHelpers "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock dependencies
	mockUserService := userservice.NewMockService(ctrl)
	mockLogger := loggerfrolfbot.NoOpLogger
	mockTracer := noop.NewTracerProvider().Tracer("test")
	mockHelpers := mockHelpers.NewMockHelpers(ctrl)
	mockMetrics := mocks.NewMockUserMetrics(ctrl)

	// Test with all dependencies
	handlers := NewUserHandlers(mockUserService, mockLogger, mockTracer, mockHelpers, mockMetrics)
	if handlers == nil {
		t.Fatal("Expected non-nil handlers")
	}

	// Test with nil dependencies
	handlersNil := NewUserHandlers(nil, nil, nil, nil, nil)
	if handlersNil == nil {
		t.Fatal("Expected non-nil handlers even with nil dependencies")
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
		setup   func(a *args)
	}{
		{
			name: "Successful execution",
			args: func(ctrl *gomock.Controller) args {
				mockLogger := loggerfrolfbot.NoOpLogger
				mockMetrics := mocks.NewMockUserMetrics(ctrl)
				mockTracer := noop.NewTracerProvider().Tracer("test")
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
				mockMetrics := a.metrics.(*mocks.MockUserMetrics)
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
				mockMetrics := a.metrics.(*mocks.MockUserMetrics)
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
