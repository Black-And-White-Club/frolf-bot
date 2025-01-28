package logging

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// Logger defines the interface for structured logging. Extend this as needed.
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Warn(msg string, args ...interface{}) // Optional, depending on your logging library
}

// LogWithMetadataCtx logs a message with structured metadata, context, and message details.
// Supports various log levels (info, error, debug, warn).
func LogWithMetadataCtx(ctx context.Context, logger Logger, msg *message.Message, level string, logMessage string, metadata map[string]interface{}) {
	correlationID := middleware.MessageCorrelationID(msg)
	fields := []interface{}{"correlation_id", correlationID}

	// Extract additional metadata from context
	if requestID, ok := ctx.Value("request_id").(string); ok && requestID != "" {
		fields = append(fields, "request_id", requestID)
	}

	// Extract "caused_by" metadata from the message
	if causedBy := msg.Metadata.Get("caused_by"); causedBy != "" {
		fields = append(fields, "caused_by", causedBy)
	}

	// Add provided metadata
	for key, value := range metadata {
		if key != "" && value != nil {
			fields = append(fields, key, value)
		}
	}

	// Log at the specified level
	switch level {
	case "info":
		logger.Info(logMessage, fields...)
	case "error":
		logger.Error(logMessage, fields...)
	case "debug":
		logger.Debug(logMessage, fields...)
	case "warn":
		logger.Warn(logMessage, fields...)
	default:
		logger.Info(logMessage, fields...) // Default to info
	}
}

// LogInfoWithMetadata logs an info-level message with metadata.
// Convenience wrapper for LogWithMetadataCtx.
func LogInfoWithMetadata(ctx context.Context, logger Logger, msg *message.Message, logMessage string, metadata map[string]interface{}) {
	LogWithMetadataCtx(ctx, logger, msg, "info", logMessage, metadata)
}

// LogErrorWithMetadata logs an error-level message with metadata.
// Convenience wrapper for LogWithMetadataCtx.
func LogErrorWithMetadata(ctx context.Context, logger Logger, msg *message.Message, logMessage string, metadata map[string]interface{}) {
	LogWithMetadataCtx(ctx, logger, msg, "error", logMessage, metadata)
}

// LogDebugWithMetadata logs a debug-level message with metadata.
// Convenience wrapper for LogWithMetadataCtx.
func LogDebugWithMetadata(ctx context.Context, logger Logger, msg *message.Message, logMessage string, metadata map[string]interface{}) {
	LogWithMetadataCtx(ctx, logger, msg, "debug", logMessage, metadata)
}

// LogWarnWithMetadata logs a warning-level message with metadata.
// Convenience wrapper for LogWithMetadataCtx.
func LogWarnWithMetadata(ctx context.Context, logger Logger, msg *message.Message, logMessage string, metadata map[string]interface{}) {
	LogWithMetadataCtx(ctx, logger, msg, "warn", logMessage, metadata)
}
