package natsutil

import (
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"
)

// ConnectionConfig holds the configuration for the NATS connection pool.
type ConnectionConfig struct {
	URL      string
	PoolSize int
}

type ConnectionPool struct {
	pool     chan *nats.Conn
	poolSize int // Add poolSize to the ConnectionPool struct
	mu       sync.Mutex
}

var (
	connPool *ConnectionPool
	once     sync.Once
)

// GetConnection retrieves a connection from the pool.
func GetConnection(config ConnectionConfig) (*nats.Conn, error) {
	once.Do(func() {
		pool := make(chan *nats.Conn, config.PoolSize)
		for i := 0; i < config.PoolSize; i++ {
			conn, err := nats.Connect(config.URL, nats.Name("App Service"))
			if err != nil {
				panic(fmt.Errorf("failed to connect to NATS: %w", err))
			}
			pool <- conn
		}
		connPool = &ConnectionPool{
			pool:     pool,
			poolSize: config.PoolSize, // Store poolSize in the struct
			mu:       sync.Mutex{},
		}
	})

	conn := <-connPool.pool
	return conn, nil
}

// ReleaseConnection returns a connection to the pool.
func ReleaseConnection(conn *nats.Conn) {
	connPool.mu.Lock()
	defer connPool.mu.Unlock()
	connPool.pool <- conn
}

// Close closes all connections in the pool.
func Close() {
	connPool.mu.Lock()
	defer connPool.mu.Unlock()
	close(connPool.pool)
	for i := 0; i < connPool.poolSize; i++ {
		conn := <-connPool.pool
		conn.Close() // Correct: conn.Close() does not return an error
	}
}
