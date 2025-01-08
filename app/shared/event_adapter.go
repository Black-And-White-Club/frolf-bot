// shared/event_adapter.go
package shared

import "context"

// EventAdapterInterface defines the interface for event adapters.
type EventAdapterInterface interface {
	Publish(ctx context.Context, eventType EventType, payload []byte, metadata map[string]string) error
	Subscribe(ctx context.Context, eventType EventType, queueGroup string, handler func(payload []byte, metadata map[string]string) error) error
	CreateStream(ctx context.Context, eventType EventType, subjects []string) error
	Close() error
}
