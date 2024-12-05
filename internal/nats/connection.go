package nats

import (
	"fmt"
	"log"
	"sync"

	"github.com/nats-io/nats.go"
)

// NatsConnectionPool manages a pool of NATS connections.
type NatsConnectionPool struct {
	pool     chan *nats.Conn
	url      string
	poolSize int
	mu       sync.Mutex
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
		log.Println("Getting connection from pool, Status:", conn.Status()) // Log connection status
		return conn, nil
	default:
		// Instead of returning an error, try creating a new connection here
		conn, err := nats.Connect(ncp.url, nats.Name("App Service"))
		if err != nil {
			log.Printf("Error creating new NATS connection: %v", err)
			return nil, fmt.Errorf("no available connections in pool, and failed to create new connection: %w", err)
		}
		log.Println("Created a new NATS connection")
		return conn, nil
	}
}

// ReleaseConnection returns a connection to the pool.
func (ncp *NatsConnectionPool) ReleaseConnection(conn *nats.Conn) {
	ncp.mu.Lock()
	defer ncp.mu.Unlock()
	log.Println("Releasing connection to pool, Status:", conn.Status()) // Log connection status
	ncp.pool <- conn
	log.Println("Releasing connection to pool")
}

// Close closes all connections in the pool.
func (ncp *NatsConnectionPool) Close() {
	close(ncp.pool)
	for conn := range ncp.pool {
		conn.Close()
	}
}
