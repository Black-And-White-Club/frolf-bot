package testutils

import (
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
)

// MessageCapture captures messages flowing through handlers for test verification
type MessageCapture struct {
	messages map[string][]*message.Message
	mutex    sync.RWMutex
	filters  map[string]bool // topics to capture
}

// NewMessageCapture creates a new message capture instance
func NewMessageCapture(topicsToCapture ...string) *MessageCapture {
	filters := make(map[string]bool)
	for _, topic := range topicsToCapture {
		filters[topic] = true
	}

	return &MessageCapture{
		messages: make(map[string][]*message.Message),
		filters:  filters,
	}
}

// CaptureHandler creates a handler that captures messages for a specific topic
func (mc *MessageCapture) CaptureHandler(topic string) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		// Capture the message if it matches our filters
		if len(mc.filters) == 0 || mc.filters[topic] {
			mc.mutex.Lock()
			mc.messages[topic] = append(mc.messages[topic], msg)
			mc.mutex.Unlock()
		}

		return nil, nil
	}
}

// GetMessages returns captured messages for a specific topic
func (mc *MessageCapture) GetMessages(topic string) []*message.Message {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	// Return a copy to avoid race conditions
	msgs := make([]*message.Message, len(mc.messages[topic]))
	copy(msgs, mc.messages[topic])
	return msgs
}

// Clear clears all captured messages
func (mc *MessageCapture) Clear() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.messages = make(map[string][]*message.Message)
}

// WaitForMessages waits for a specific number of messages on a topic with timeout
func (mc *MessageCapture) WaitForMessages(topic string, expectedCount int, timeout time.Duration) bool {
	interval := 1 * time.Millisecond
	maxInterval := 50 * time.Millisecond
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if len(mc.GetMessages(topic)) >= expectedCount {
			return true
		}

		time.Sleep(interval)
		interval *= 2
		if interval > maxInterval {
			interval = maxInterval
		}
	}
	return false
}

func (mc *MessageCapture) ClearOldMessages(olderThan time.Duration) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	cutoff := time.Now().Add(-olderThan)

	for topic, messages := range mc.messages {
		filtered := make([]*message.Message, 0, len(messages))
		for _, msg := range messages {
			// Keep recent messages, clear old ones
			if msg.Metadata.Get("timestamp") != "" {
				if timestamp, err := time.Parse(time.RFC3339, msg.Metadata.Get("timestamp")); err == nil {
					if timestamp.After(cutoff) {
						filtered = append(filtered, msg)
					}
				} else {
					filtered = append(filtered, msg) // Keep if can't parse timestamp
				}
			} else {
				filtered = append(filtered, msg) // Keep if no timestamp
			}
		}
		mc.messages[topic] = filtered
	}
}

func (mc *MessageCapture) WaitForMessageType(topic string, messageType string, timeout time.Duration) *message.Message {
	deadline := time.Now().Add(timeout)
	interval := 1 * time.Millisecond
	maxInterval := 50 * time.Millisecond

	for time.Now().Before(deadline) {
		messages := mc.GetMessages(topic)
		for _, msg := range messages {
			if msg.Metadata.Get("type") == messageType {
				return msg
			}
		}

		time.Sleep(interval)
		interval *= 2
		if interval > maxInterval {
			interval = maxInterval
		}
	}
	return nil
}
