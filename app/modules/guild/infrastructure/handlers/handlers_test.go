package guildhandlers

import (
	"testing"

	mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	guildmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewGuildHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := guildmocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &guildmetrics.NoOpMetrics{}

	handlers := NewGuildHandlers(mockService, logger, tracer, mockHelpers, metrics)

	if handlers == nil {
		t.Fatal("NewGuildHandlers returned nil")
	}

	if handlers.guildService != mockService {
		t.Error("guildService not set correctly")
	}
	if handlers.logger != logger {
		t.Error("logger not set correctly")
	}
	if handlers.tracer != tracer {
		t.Error("tracer not set correctly")
	}
	if handlers.helpers != mockHelpers {
		t.Error("helpers not set correctly")
	}
	if handlers.metrics != metrics {
		t.Error("metrics not set correctly")
	}
	if handlers.handlerWrapper == nil {
		t.Error("handlerWrapper not set")
	}
}
