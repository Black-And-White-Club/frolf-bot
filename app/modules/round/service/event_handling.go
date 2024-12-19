package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/events"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/nats-io/nats.go"
)

// RoundService handles round-related logic.
type RoundService struct {
	RoundDB   rounddb.RoundDB
	JS        nats.JetStreamContext
	Publisher nats.JetStreamContext
}

// NewRoundService creates a new RoundService.
func NewRoundService(js nats.JetStreamContext, db rounddb.RoundDB) *RoundService {
	return &RoundService{
		RoundDB:   db,
		JS:        js,
		Publisher: js,
	}
}

// getTagNumber retrieves the tag number for a participant from the leaderboard module.
func (s *RoundService) getTagNumber(ctx context.Context, discordID string) (*int, error) {
	// 1. Publish GetTagNumberRequest event
	req := roundevents.GetTagNumberRequest{
		DiscordID: discordID,
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GetTagNumberRequest: %w", err)
	}
	correlationID := watermill.NewUUID()

	// 2. Publish the event with the correlation ID
	_, err = s.Publisher.Publish(roundevents.GetTagNumberRequestSubject, reqData, nats.MsgId(correlationID))
	if err != nil {
		return nil, fmt.Errorf("failed to publish GetTagNumberRequest: %w", err)
	}

	// 3. Subscribe to GetTagNumberResponseSubject with the correlation ID
	sub, err := s.JS.QueueSubscribeSync(roundevents.GetTagNumberResponseSubject, correlationID, nats.BindStream(roundevents.LeaderboardStream))
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to GetTagNumberResponse: %w", err)
	}
	defer sub.Unsubscribe()

	// 4. Receive the response
	msg, err := sub.NextMsgWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to receive GetTagNumberResponse: %w", err)
	}

	// 5. Unmarshal the response
	var resp roundevents.GetTagNumberResponseEvent
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GetTagNumberResponse: %w", err)
	}

	if resp.TagNumber == 0 { // Check if the tag number is 0 (meaning no tag)
		return nil, nil // No tag number found
	}

	return &resp.TagNumber, nil
}

// getUserRole retrieves the role of a user from the user module.
func (s *RoundService) getUserRole(ctx context.Context, discordID string) (string, error) { // Change return type to string
	// 1. Publish GetUserRoleRequestEvent
	req := roundevents.GetUserRoleRequestEvent{
		DiscordID: discordID,
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal GetUserRoleRequestEvent: %w", err)
	}
	correlationID := watermill.NewUUID()

	// 2. Publish the event with the correlation ID
	_, err = s.Publisher.Publish(roundevents.GetUserRoleRequestSubject, reqData, nats.MsgId(correlationID))
	if err != nil {
		return "", fmt.Errorf("failed to publish GetUserRoleRequestEvent: %w", err)
	}

	// 3. Subscribe to GetUserRoleResponseSubject with the correlation ID
	sub, err := s.JS.QueueSubscribeSync(roundevents.GetUserRoleResponseSubject, correlationID, nats.BindStream(roundevents.UserStream))
	if err != nil {
		return "", fmt.Errorf("failed to subscribe to GetUserRoleResponseEvent: %w", err)
	}
	defer sub.Unsubscribe()

	// 4. Receive the response
	msg, err := sub.NextMsgWithContext(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to receive GetUserRoleResponseEvent: %w", err)
	}

	// 5. Unmarshal the response
	var resp roundevents.GetUserRoleResponseEvent
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal GetUserRoleResponseEvent: %w", err)
	}

	return resp.Role, nil // Return the role as a string
}

func (s *RoundService) scheduleRoundEvents(ctx context.Context, roundID string, startTime time.Time) error {
	oneHourBefore := startTime.Add(-1 * time.Hour)
	thirtyMinsBefore := startTime.Add(-30 * time.Minute)

	// Schedule one-hour reminder
	err := s.scheduleEvent(ctx, "round.reminder", &roundevents.RoundReminderEvent{
		RoundID:      roundID,
		ReminderType: "one_hour",
	}, oneHourBefore)
	if err != nil {
		return fmt.Errorf("failed to schedule one-hour reminder: %w", err)
	}

	// Schedule 30-minute reminder
	err = s.scheduleEvent(ctx, "round.reminder", &roundevents.RoundReminderEvent{
		RoundID:      roundID,
		ReminderType: "thirty_minutes",
	}, thirtyMinsBefore)
	if err != nil {
		return fmt.Errorf("failed to schedule thirty-minutes reminder: %w", err)
	}

	// Schedule round start
	err = s.scheduleEvent(ctx, "round.started", &roundevents.RoundStartedEvent{
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

	// 2. Create a NATS message with the delivery time header
	msg := nats.NewMsg(subject)
	msg.Data = data
	msg.Header.Set("NatsDeliveryTime", deliveryTime.UTC().Format(time.RFC3339)) // Corrected line

	// 3. Publish the message to JetStream
	_, err = s.Publisher.PublishMsg(msg)
	if err != nil {
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

	_, err = s.Publisher.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}
