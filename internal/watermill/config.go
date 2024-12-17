package watermillutil

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// NewRouter creates a new Watermill router with NATS JetStream.
func NewRouter(natsURL string, logger watermill.LoggerAdapter) (*message.Router, *PubSub, error) {
	logger.Info("Creating Watermill PubSub instance", nil)
	pubsub, err := NewPubSub(natsURL, logger, nil, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Watermill pubsub: %w", err)
	}
	logger.Info("Created Watermill PubSub instance", nil)

	logger.Info("Creating Watermill Router", nil)
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Watermill router: %w", err)
	}
	logger.Info("Created Watermill Router", nil)

	router.AddMiddleware(
		middleware.Recoverer,
	)

	return router, pubsub, nil
}
