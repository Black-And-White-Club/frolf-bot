package watermillutil

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

func NewRouter(natsURL string, logger watermill.LoggerAdapter) (*message.Router, *PubSub, error) {
	pubsub, err := NewPubSub(natsURL, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Watermill pubsub: %w", err)
	}

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Watermill router: %w", err)
	}

	router.AddMiddleware(
		middleware.Recoverer,
	)

	return router, pubsub, nil
}
