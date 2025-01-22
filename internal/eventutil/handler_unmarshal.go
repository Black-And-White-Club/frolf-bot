package eventutil

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// UnmarshalPayload unmarshals the event payload into the provided type.
func UnmarshalPayload[T any](msg *message.Message, logger *slog.Logger) (string, *T, error) {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)

	var payload T
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		logger.Error("Failed to unmarshal message payload",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return correlationID, nil, fmt.Errorf("failed to unmarshal message payload: %w", err)
	}

	return correlationID, &payload, nil
}
