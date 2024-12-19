package roundsubscribers

import (
	roundhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/round/handlers"
	"github.com/nats-io/nats.go"
)

// RoundSubscribers subscribes to round-related events.
type RoundSubscribers struct {
	JS       nats.JetStreamContext
	Handlers roundhandlers.RoundHandlers
}
