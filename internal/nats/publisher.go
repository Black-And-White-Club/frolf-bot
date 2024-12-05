package nats

import (
	"encoding/json"
	"fmt"
)

// Publish sends a message to a NATS subject using the connection pool.
func (ncp *NatsConnectionPool) Publish(subject string, data interface{}) error {
	// Get a connection from the pool
	conn, err := ncp.GetConnection()
	if err != nil {
		return fmt.Errorf("failed to get NATS connection from pool: %w", err)
	}
	defer ncp.ReleaseConnection(conn) // Release the connection back to the pool

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data for subject %s: %w", subject, err)
	}

	err = conn.Publish(subject, payload)
	if err != nil {
		return fmt.Errorf("failed to publish event to subject %s: %w", subject, err)
	}

	return nil
}
