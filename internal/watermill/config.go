package watermillutil

import (
	"fmt"
	"log"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

func NewRouter(natsURL string, logger watermill.LoggerAdapter) (*message.Router, *PubSub, error) {
	logger.Info("Creating Watermill PubSub instance", nil)
	pubsub, err := NewPubSub(natsURL, logger, nil, nil)
	if err != nil {
		log.Printf("Failed to create Watermill pubsub: %v", err) // Log the error
		return nil, nil, fmt.Errorf("failed to create Watermill pubsub: %w", err)
	}
	logger.Info("Created Watermill PubSub instance", nil)

	logger.Info("Creating Watermill Router", nil)
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		log.Printf("Failed to create Watermill router: %v", err) // Log the error
		return nil, nil, fmt.Errorf("failed to create Watermill router: %w", err)
	}
	logger.Info("Created Watermill Router", nil)

	router.AddMiddleware(
		middleware.Recoverer,
	)

	return router, pubsub, nil
}
