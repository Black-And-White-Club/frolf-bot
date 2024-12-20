package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/events"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// getTagNumber retrieves the tag number for a participant from the leaderboard module.
func (s *RoundService) getTagNumber(ctx context.Context, discordID string) (*int, error) {
	// 1. Create a Watermill message with a correlation ID
	correlationID := watermill.NewUUID()
	msg := message.NewMessage(correlationID, nil)

	// 2. Publish GetTagNumberRequest event
	req := roundevents.GetTagNumberRequest{
		DiscordID: discordID,
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GetTagNumberRequest: %w", err)
	}
	msg.Payload = reqData

	if err := s.Publisher.Publish(roundevents.GetTagNumberRequestSubject, msg); err != nil {
		return nil, fmt.Errorf("failed to publish GetTagNumberRequest: %w", err)
	}

	// 3. Create a subscriber for the response
	messages, err := s.Subscriber.Subscribe(ctx, roundevents.GetTagNumberResponseSubject)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to GetTagNumberResponse: %w", err)
	}

	// 4. Receive the response with the matching correlation ID
	var respMsg *message.Message
	for {
		respMsg = <-messages

		if respMsg.UUID == correlationID { // Check if the correlation IDs match
			break
		}

		// If the correlation ID doesn't match, nack the message and continue listening
		respMsg.Nack()
	}

	// 5. Unmarshal the response
	var resp roundevents.GetTagNumberResponseEvent
	if err := json.Unmarshal(respMsg.Payload, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GetTagNumberResponse: %w", err)
	}

	if resp.TagNumber == 0 { // Check if the tag number is 0 (meaning no tag)
		return nil, nil // No tag number found
	}

	return &resp.TagNumber, nil
}

// getUserRole retrieves the role of a user from the user module.
func (s *RoundService) getUserRole(ctx context.Context, discordID string) (string, error) {
	// 1. Create a Watermill message with a correlation ID
	correlationID := watermill.NewUUID()
	msg := message.NewMessage(correlationID, nil)

	// 2. Publish GetUserRoleRequestEvent
	req := roundevents.GetUserRoleRequestEvent{
		DiscordID: discordID,
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal GetUserRoleRequestEvent: %w", err)
	}
	msg.Payload = reqData

	if err := s.Publisher.Publish(roundevents.GetUserRoleRequestSubject, msg); err != nil {
		return "", fmt.Errorf("failed to publish GetUserRoleRequestEvent: %w", err)
	}

	// 3. Create a subscriber for the response
	subscriber, err := s.Subscriber.Subscribe(ctx, roundevents.GetUserRoleResponseSubject)
	if err != nil {
		return "", fmt.Errorf("failed to subscribe to GetUserRoleResponseEvent: %w", err)
	}

	// 4. Receive the response with the matching correlation ID
	var respMsg *message.Message
	for {
		respMsg = <-subscriber

		if respMsg.UUID == correlationID { // Check if the correlation IDs match
			break
		}

		// If the correlation ID doesn't match, nack the message and continue listening
		respMsg.Nack()
	}

	// 5. Unmarshal the response
	var resp roundevents.GetUserRoleResponseEvent
	if err := json.Unmarshal(respMsg.Payload, &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal GetUserRoleResponseEvent: %w", err)
	}

	return resp.Role, nil
}

func (s *RoundService) scheduleRoundEvents(ctx context.Context, roundID string, startTime time.Time) error {
	oneHourBefore := startTime.Add(-1 * time.Hour)
	thirtyMinsBefore := startTime.Add(-30 * time.Minute)

	// Schedule one-hour reminder
	err := s.scheduleEvent(ctx, roundevents.RoundReminderSubject, &roundevents.RoundReminderEvent{
		RoundID:      roundID,
		ReminderType: "one_hour",
	}, oneHourBefore)
	if err != nil {
		return fmt.Errorf("failed to schedule one-hour reminder: %w", err)
	}

	// Schedule 30-minute reminder
	err = s.scheduleEvent(ctx, roundevents.RoundReminderSubject, &roundevents.RoundReminderEvent{
		RoundID:      roundID,
		ReminderType: "thirty_minutes",
	}, thirtyMinsBefore)
	if err != nil {
		return fmt.Errorf("failed to schedule thirty-minutes reminder: %w", err)
	}

	// Schedule round start
	err = s.scheduleEvent(ctx, roundevents.RoundStartedSubject, &roundevents.RoundStartedEvent{
		RoundID: roundID,
	}, startTime)
	if err != nil {
		return fmt.Errorf("failed to schedule round start: %w", err)
	}

	return nil
}

func (s *RoundService) scheduleEvent(_ context.Context, subject string, event any, deliveryTime time.Time) error {
	// 1. Marshal the event data
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// 2. Create a Watermill message with the delivery time header
	msg := message.NewMessage(watermill.NewUUID(), data)
	msg.Metadata.Set("Nats-Delivery-Time", deliveryTime.UTC().Format(time.RFC3339))

	// 3. Publish the message to JetStream
	if err := s.Publisher.Publish(subject, msg); err != nil {
		return fmt.Errorf("failed to publish scheduled event: %w", err)
	}

	return nil
}

// publishEvent publishes an event to the given subject.
func (s *RoundService) publishEvent(_ context.Context, subject string, event any) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := s.Publisher.Publish(subject, message.NewMessage(watermill.NewUUID(), data)); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}
