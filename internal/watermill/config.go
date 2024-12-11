package watermillutil

import (
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/internal/nats"
	"github.com/ThreeDotsLabs/watermill"
	wm "github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

type Config struct {
	Nats nats.ConnectionConfig
}

func NewRouter(config Config, logger watermill.LoggerAdapter) (*wm.Router, *PubSub, error) {
	pub, err := nats.NewWatermillPublisher(config.Nats, logger) // Use NewWatermillPublisher
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Watermill publisher: %w", err)
	}

	sub, err := nats.NewWatermillSubscriber(config.Nats, logger) // Use NewWatermillSubscriber
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Watermill subscriber: %w", err)
	}

	router, err := wm.NewRouter(wm.RouterConfig{}, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Watermill router: %w", err)
	}

	router.AddMiddleware(
		middleware.Recoverer,
	)

	pubsub := &PubSub{
		publisher:  pub,
		subscriber: sub,
	}

	return router, pubsub, nil
}
