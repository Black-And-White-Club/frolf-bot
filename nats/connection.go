package nats

import (
	"fmt"
	"log"
	"sync"

	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/jetstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go"
)

// NatsConnectionPool manages a pool of NATS connections.
type NatsConnectionPool struct {
	pool     chan *nats.Conn
	url      string
	poolSize int
	mu       sync.Mutex
}

func (ncp *NatsConnectionPool) GetURL() string {
	return ncp.url
}

// NewNatsConnectionPool creates a new NatsConnectionPool.
func NewNatsConnectionPool(url string, poolSize int) (*NatsConnectionPool, error) {
	pool := make(chan *nats.Conn, poolSize)
	for i := 0; i < poolSize; i++ {
		conn, err := nats.Connect(url, nats.Name("App Service"))
		if err != nil {
			return nil, fmt.Errorf("failed to connect to NATS: %w", err)
		}
		log.Printf("Created NATS connection %d\n", i+1)
		pool <- conn
	}
	return &NatsConnectionPool{
		pool:     pool,
		url:      url,
		poolSize: poolSize,
		mu:       sync.Mutex{},
	}, nil
}

// GetConnection retrieves a connection from the pool.
func (ncp *NatsConnectionPool) GetConnection() (*nats.Conn, error) {
	ncp.mu.Lock()
	defer ncp.mu.Unlock()
	log.Printf("Available connections in pool: %d", len(ncp.pool))
	select {
	case conn := <-ncp.pool:
		status := conn.Status()
		log.Println("Getting connection from pool, Status:", status)
		if status != nats.CONNECTED {
			log.Printf("Warning: Retrieved connection is not CONNECTED. Status: %s", status)
		}
		return conn, nil
	default:
		log.Println("No available connections in pool. Creating a new connection...")
		conn, err := nats.Connect(ncp.url, nats.Name("App Service"))
		if err != nil {
			log.Printf("Error creating new NATS connection: %v", err)
			return nil, fmt.Errorf("no available connections in pool: %w", err)
		}
		log.Println("Created a new NATS connection")
		return conn, nil
	}
}

// ReleaseConnection returns a connection to the pool.
func (ncp *NatsConnectionPool) ReleaseConnection(conn *nats.Conn) {
	ncp.mu.Lock()
	defer ncp.mu.Unlock()
	log.Println("Releasing connection to pool, Status:", conn.Status())
	ncp.pool <- conn
}

// Close closes all connections in the pool.
func (ncp *NatsConnectionPool) Close() {
	close(ncp.pool)
	for conn := range ncp.pool {
		conn.Close()
	}
}

// Publish publishes a message to the given topic.
func (ncp *NatsConnectionPool) Publish(topic string, messages ...*message.Message) error {
	conn, err := ncp.GetConnection()
	if err != nil {
		return fmt.Errorf("failed to get NATS connection: %w", err)
	}
	defer ncp.ReleaseConnection(conn)

	publisher, err := jetstream.NewPublisher(
		jetstream.PublisherConfig{}, // Only the config is needed
	)
	if err != nil {
		return fmt.Errorf("failed to create NATS publisher: %w", err)
	}

	for _, msg := range messages {
		if err := publisher.Publish(topic, msg); err != nil {
			return fmt.Errorf("failed to publish message: %w", err)
		}
	}

	return nil
}
