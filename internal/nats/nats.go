package natsutil

import (
	"context"
	"fmt"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

// Config holds the configuration for the NATS connection and Watermill integration.
type Config struct {
	URL             string
	PoolSize        int
	JetStreamConfig nats.JetStreamConfig
}

// NatsPublisher embeds the Watermill publisher and manages the connection pool.
type NatsPublisher struct {
	publisher message.Publisher
	pool      chan *nc.Conn
	config    Config
	mu        sync.Mutex
}

// NewPublisher creates a new NatsPublisher.
func NewPublisher(config Config, logger watermill.LoggerAdapter) (*NatsPublisher, error) {
	pool := make(chan *nc.Conn, config.PoolSize)
	for i := 0; i < config.PoolSize; i++ {
		conn, err := nc.Connect(config.URL, nc.Name("Watermill NATS Publisher"))
		if err != nil {
			return nil, fmt.Errorf("failed to connect to NATS: %w", err)
		}
		pool <- conn
	}

	config.JetStreamConfig.AutoProvision = true // Ensure auto-provisioning is enabled
	pub, err := nats.NewPublisher(nats.PublisherConfig{
		Conn:      <-pool,
		JetStream: config.JetStreamConfig, // Use the provided JetStream config
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill publisher: %w", err)
	}

	return &NatsPublisher{publisher: pub, pool: pool, config: config}, nil
}

func (p *NatsPublisher) Publish(topic string, messages ...*message.Message) error {
	return p.publisher.Publish(topic, messages...)
}

func (p *NatsPublisher) Close() error {
	close(p.pool)
	for i := 0; i < p.config.PoolSize; i++ {
		conn := <-p.pool
		conn.Close()
	}
	return nil
}

// NatsSubscriber embeds the Watermill subscriber and manages the connection pool.
type NatsSubscriber struct {
	subscriber message.Subscriber
	pool       chan *nc.Conn
	config     Config
	mu         sync.Mutex
}

// NewSubscriber creates a new NatsSubscriber.
func NewSubscriber(config Config, logger watermill.LoggerAdapter) (*NatsSubscriber, error) {
	pool := make(chan *nc.Conn, config.PoolSize)
	for i := 0; i < config.PoolSize; i++ {
		conn, err := nc.Connect(config.URL, nc.Name("Watermill NATS Subscriber"))
		if err != nil {
			return nil, fmt.Errorf("failed to connect to NATS: %w", err)
		}
		pool <- conn
	}

	config.JetStreamConfig.AutoProvision = true // Ensure auto-provisioning is enabled
	sub, err := nats.NewSubscriber(nats.SubscriberConfig{
		Conn:      <-pool,
		JetStream: config.JetStreamConfig, // Use the provided JetStream config
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill subscriber: %w", err)
	}

	return &NatsSubscriber{subscriber: sub, pool: pool, config: config}, nil
}

func (s *NatsSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return s.subscriber.Subscribe(ctx, topic)
}

func (s *NatsSubscriber) Close() error {
	close(s.pool)
	for i := 0; i < s.config.PoolSize; i++ {
		conn := <-s.pool
		conn.Close()
	}
	return nil
}
