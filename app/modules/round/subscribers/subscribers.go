package roundsubscribers

import (
	roundhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/round/handlers"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// RoundSubscribers subscribes to round-related events.
type RoundSubscribers struct {
	Subscriber message.Subscriber
	logger     watermill.LoggerAdapter
	Handlers   roundhandlers.RoundHandlers
}
