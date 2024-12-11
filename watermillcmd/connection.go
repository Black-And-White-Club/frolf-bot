// watermillcmd/connection.go
package watermillcmd

import (
	"sync"

	natsjetstream "github.com/Black-And-White-Club/tcr-bot/nats"
	"github.com/nats-io/nats.go"
)

var (
	// Create a connection pool (initialize it elsewhere in your application)
	natsConnectionPool *natsjetstream.NatsConnectionPool
	once               sync.Once
)

// GetConnection retrieves a connection from the pool.
func GetConnection() (*nats.Conn, error) {
	// Initialize the connection pool only once
	once.Do(func() {
		var err error
		natsConnectionPool, err = natsjetstream.NewNatsConnectionPool("your_nats_url", 10) // Replace with your NATS URL and pool size
		if err != nil {
			panic(err) // Handle the error appropriately
		}
	})

	return natsConnectionPool.GetConnection()
}
