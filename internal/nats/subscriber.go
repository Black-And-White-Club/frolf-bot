package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
)

// Subscribe sets up a NATS subscription with a handler function using the connection pool.
func (ncp *NatsConnectionPool) Subscribe(ctx context.Context, subject string, handler func(msg *nats.Msg)) error {
	// Get a connection from the pool
	conn, err := ncp.GetConnection()
	if err != nil {
		return fmt.Errorf("failed to get NATS connection from pool: %w", err)
	}
	defer ncp.ReleaseConnection(conn) // Release the connection back to the pool

	_, err = conn.Subscribe(subject, handler)
	if err != nil {
		return fmt.Errorf("failed to subscribe to subject %s: %w", subject, err)
	}

	log.Printf("Subscribed to subject %s", subject)
	return nil
}

// Request sends a request and waits for a reply using the connection pool.
func (ncp *NatsConnectionPool) Request(ctx context.Context, subject string, data interface{}, timeoutSeconds int) ([]byte, error) {
	// Get a connection from the pool
	conn, err := ncp.GetConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to get NATS connection from pool: %w", err)
	}
	defer ncp.ReleaseConnection(conn) // Release the connection back to the pool

	payload, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data for subject %s: %w", subject, err)
	}

	// Use RequestWithContext with the provided context
	msg, err := conn.RequestWithContext(ctx, subject, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to request on subject %s: %w", subject, err)
	}

	return msg.Data, nil
}
