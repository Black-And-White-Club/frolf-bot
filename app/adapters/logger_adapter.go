package adapters

import (
	"github.com/Black-And-White-Club/tcr-bot/app/types"

	"github.com/ThreeDotsLabs/watermill"
)

// AppLoggerAdapter adapts the types.LoggerAdapter to the watermill.LoggerAdapter interface.
type AppLoggerAdapter struct {
	logger types.LoggerAdapter
}

// NewAppLoggerAdapter creates a new AppLoggerAdapter.
func NewAppLoggerAdapter(logger types.LoggerAdapter) *AppLoggerAdapter {
	return &AppLoggerAdapter{logger: logger}
}

// Error logs an error message.
func (l *AppLoggerAdapter) Error(msg string, err error, fields watermill.LogFields) {
	l.logger.Error(msg, err, types.LogFields(fields))
}

// Info logs an info message.
func (l *AppLoggerAdapter) Info(msg string, fields watermill.LogFields) {
	l.logger.Info(msg, types.LogFields(fields))
}

// Debug logs a debug message.
func (l *AppLoggerAdapter) Debug(msg string, fields watermill.LogFields) {
	l.logger.Debug(msg, types.LogFields(fields))
}

// Trace logs a trace message.
func (l *AppLoggerAdapter) Trace(msg string, fields watermill.LogFields) {
	l.logger.Trace(msg, types.LogFields(fields))
}

// With returns a new logger with the given fields.
func (l *AppLoggerAdapter) With(fields watermill.LogFields) watermill.LoggerAdapter {
	return &AppLoggerAdapter{logger: l.logger.With(types.LogFields(fields))}
}
