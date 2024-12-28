package roundservice

import (
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// RoundService handles round-related logic.
type RoundService struct {
	RoundDB    rounddb.RoundDB
	Publisher  message.Publisher
	Subscriber message.Subscriber
	logger     watermill.LoggerAdapter
}

// NewRoundService creates a new RoundService.
func NewRoundService(publisher message.Publisher, subscriber message.Subscriber, db rounddb.RoundDB, logger watermill.LoggerAdapter) Service {
	return &RoundService{
		RoundDB:    db,
		Publisher:  publisher,
		Subscriber: subscriber,
		logger:     logger,
	}
}
