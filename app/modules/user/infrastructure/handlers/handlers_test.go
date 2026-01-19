package userhandlers

import (
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"

	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application/mocks"
)

func TestNewUserHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock dependencies
	mockUserService := userservice.NewMockService(ctrl)
	mockLogger := loggerfrolfbot.NoOpLogger
	mockTracer := noop.NewTracerProvider().Tracer("test")

	// Test with all dependencies
	handlers := NewUserHandlers(mockUserService, mockLogger, mockTracer, nil, nil)
	if handlers == nil {
		t.Fatal("Expected non-nil handlers")
	}

	// Test with nil dependencies
	handlersNil := NewUserHandlers(nil, nil, nil, nil, nil)
	if handlersNil == nil {
		t.Fatal("Expected non-nil handlers even with nil dependencies")
	}
}
