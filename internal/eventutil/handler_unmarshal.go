package eventutil

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// UnmarshalPayload unmarshals the message payload into the provided type.
// It also extracts the correlation ID for logging.
func UnmarshalPayload[T any](msg *message.Message, logger *slog.Logger) (string, T, error) {
	var payload T
	correlationID := middleware.MessageCorrelationID(msg)

	logger.Debug("Attempting to unmarshal payload",
		slog.String("correlation_id", correlationID),
		slog.String("message_uuid", msg.UUID),
		slog.String("payload", string(msg.Payload)),
		slog.String("type", fmt.Sprintf("%T", payload)), // Log the type
	)

	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		logger.Error("Failed to unmarshal message payload",
			slog.String("correlation_id", correlationID),
			slog.String("message_uuid", msg.UUID),
			slog.Any("error", err),
		)
		return correlationID, payload, err
	}

	logger.Debug("Payload unmarshalled successfully",
		slog.String("correlation_id", correlationID),
		slog.String("message_uuid", msg.UUID),
	)

	return correlationID, payload, nil
}
