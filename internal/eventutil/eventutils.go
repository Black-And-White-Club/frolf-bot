// pkg/eventutil/eventutil.go

package eventutil

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

type EventUtil interface {
	PropagateMetadata(srcMsg *message.Message, dstMsg *message.Message)
}

type EventUtilImpl struct{}

// NewEventUtil creates a new EventUtil.
func NewEventUtil() EventUtil {
	return &EventUtilImpl{}
}

// PropagateMetadata copies the correlation ID from the source message to the destination message.
func (e *EventUtilImpl) PropagateMetadata(srcMsg *message.Message, dstMsg *message.Message) {
	correlationID := srcMsg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	if correlationID != "" {
		dstMsg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)
	}
}
