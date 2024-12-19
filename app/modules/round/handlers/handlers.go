package roundhandlers

import (
	roundservice "github.com/Black-And-White-Club/tcr-bot/app/modules/round/service"
	"github.com/ThreeDotsLabs/watermill/message"
)

// RoundHandlers handles round-related events.
type RoundHandlers struct {
	RoundService roundservice.Service
	Publisher    message.Publisher
}
