package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

const defaultGetTagNumberTimeout = 5 * time.Second // Define a reasonable default timeout

func (s *RoundService) getTagNumber(ctx context.Context, discordID string, timeout ...time.Duration) (*int, error) {
	// Determine the effective timeout
	effectiveTimeout := defaultGetTagNumberTimeout
	if len(timeout) > 0 {
		effectiveTimeout = timeout[0]
	}

	// Create a context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, effectiveTimeout)
	defer cancel()

	// 1. Publish the GetTagNumberRequest event
	req := roundevents.GetTagNumberRequestPayload{
		DiscordID: discordID,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GetTagNumberRequest: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), reqData)

	if err := s.eventBus.Publish(ctxWithTimeout, roundevents.GetTagNumberRequest, msg); err != nil {
		return nil, fmt.Errorf("failed to publish GetTagNumberRequest: %w", err)
	}

	// 2. Subscribe to the GetTagNumberResponse
	responseChan := make(chan roundevents.GetTagNumberResponsePayload, 1) // Buffered to prevent blocking
	defer close(responseChan)

	err = s.eventBus.Subscribe(ctxWithTimeout, roundevents.LeaderboardStreamName, roundevents.GetTagNumberResponse, func(ctx context.Context, msg *message.Message) error {
		var resp roundevents.GetTagNumberResponsePayload
		if err := json.Unmarshal(msg.Payload, &resp); err != nil {
			return fmt.Errorf("failed to unmarshal GetTagNumberResponse: %w", err)
		}
		select {
		case responseChan <- resp:
		default: // Avoid blocking in case of a rare double-send
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to GetTagNumberResponse: %w", err)
	}

	// 3. Wait for the response or timeout
	select {
	case resp := <-responseChan:
		if resp.Error != "" {
			return nil, fmt.Errorf("error from GetTagNumberResponse: %s", resp.Error)
		}
		if resp.TagNumber == 0 {
			return nil, nil // TagNumber is zero, return nil as per the expected behavior
		}
		return &resp.TagNumber, nil
	case <-ctxWithTimeout.Done():
		// Use the exact error message expected by the test
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

// getUserRole retrieves the role of a user from the user module.
func (s *RoundService) getUserRole(ctx context.Context, discordID string) (string, error) {
	// 1. Publish GetUserRoleRequest event
	req := roundevents.GetUserRoleRequestPayload{
		DiscordID: discordID,
	}

	// Create a Watermill message
	msg := message.NewMessage(watermill.NewUUID(), nil)

	// Marshal the request payload into the message
	reqData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal GetUserRoleRequest: %w", err)
	}
	msg.Payload = reqData

	if err := s.eventBus.Publish(ctx, roundevents.GetUserRoleRequest, msg); err != nil {
		return "", fmt.Errorf("failed to publish GetUserRoleRequest: %w", err)
	}

	// 2. Subscribe to the response (using a separate subscriber)
	responseChan := make(chan roundevents.GetUserRoleResponsePayload)
	err = s.eventBus.Subscribe(ctx, roundevents.UserStreamName, roundevents.GetUserRoleResponse, func(ctx context.Context, msg *message.Message) error {
		var resp roundevents.GetUserRoleResponsePayload
		if err := json.Unmarshal(msg.Payload, &resp); err != nil {
			return fmt.Errorf("failed to unmarshal GetUserRoleResponse: %w", err)
		}
		responseChan <- resp
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to subscribe to GetUserRoleResponse: %w", err)
	}

	// 3. Receive the response (with timeout)
	var resp roundevents.GetUserRoleResponsePayload
	select {
	case resp = <-responseChan:
	case <-time.After(5 * time.Second):
		return "", fmt.Errorf("timeout waiting for response")
	}

	if resp.Error != "" {
		return "", fmt.Errorf("error getting user role: %s", resp.Error)
	}

	return resp.Role, nil
}

func (s *RoundService) scheduleRoundEvents(ctx context.Context, roundID string, startTime time.Time) error {
	oneHourBefore := startTime.Add(-1 * time.Hour)
	thirtyMinsBefore := startTime.Add(-30 * time.Minute)

	// Schedule one-hour reminder
	err := s.scheduleEvent(ctx, roundevents.RoundReminder, &roundevents.RoundReminderPayload{
		RoundID:      roundID,
		ReminderType: "one_hour",
	}, oneHourBefore)
	if err != nil {
		return fmt.Errorf("failed to schedule one-hour reminder: %w", err)
	}

	// Schedule 30-minute reminder
	err = s.scheduleEvent(ctx, roundevents.RoundReminder, &roundevents.RoundReminderPayload{
		RoundID:      roundID,
		ReminderType: "thirty_minutes",
	}, thirtyMinsBefore)
	if err != nil {
		return fmt.Errorf("failed to schedule thirty-minutes reminder: %w", err)
	}

	// Schedule round start
	err = s.scheduleEvent(ctx, roundevents.RoundStarted, &roundevents.RoundStartedPayload{
		RoundID: roundID,
	}, startTime)
	if err != nil {
		return fmt.Errorf("failed to schedule round start: %w", err)
	}

	return nil
}

func (s *RoundService) scheduleEvent(ctx context.Context, subject string, event any, deliveryTime time.Time) error {
	// 1. Marshal the event data
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// 2. Create a Watermill message with the delivery time header
	msg := message.NewMessage(watermill.NewUUID(), data)
	msg.Metadata.Set("Nats-Delivery-Time", deliveryTime.UTC().Format(time.RFC3339))

	// 3. Publish the message
	return s.eventBus.Publish(ctx, subject, msg) // Directly return the error from Publish
}

// publishEvent publishes an event to the given subject.
func (s *RoundService) publishEvent(ctx context.Context, subject string, event any) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), data)

	if err := s.eventBus.Publish(ctx, subject, msg); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}
