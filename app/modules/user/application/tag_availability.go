package userservice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"log/slog"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// TagAvailabilityChecker interface (exported)
type TagAvailabilityChecker interface {
	CheckTagAvailability(ctx context.Context, tagNumber int) (bool, error)
}

// checkTagAvailability checks if the tag is available.
func (s *UserServiceImpl) checkTagAvailability(ctx context.Context, tagNumber int) (bool, error) {
	s.logger.Info("checkTagAvailability", slog.Int("tag_number", tagNumber))

	const defaultTimeout = 5 * time.Second
	ctxWithTimeout, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	correlationID := watermill.NewUUID()
	responseChan := make(chan userevents.CheckTagAvailabilityResponsePayload)

	// Prepare the request payload
	requestPayload, err := json.Marshal(userevents.CheckTagAvailabilityRequestPayload{
		TagNumber: tagNumber,
	})
	if err != nil {
		return false, fmt.Errorf("failed to marshal CheckTagAvailabilityRequestPayload: %w", err)
	}

	// Create the request message
	requestMsg := message.NewMessage(watermill.NewUUID(), requestPayload)
	requestMsg.SetContext(ctxWithTimeout)
	requestMsg.Metadata.Set("correlation_id", correlationID)
	requestMsg.Metadata.Set("subject", userevents.CheckTagAvailabilityRequest)
	requestMsg.Metadata.Set("Reply-To", userevents.CheckTagAvailabilityResponse) // Set Reply-To metadata

	// Subscribe to the response on the LeaderboardStreamName
	err = s.eventBus.Subscribe(ctxWithTimeout, userevents.LeaderboardStreamName, userevents.CheckTagAvailabilityResponse, func(ctx context.Context, msg *message.Message) error {
		var responsePayload userevents.CheckTagAvailabilityResponsePayload
		if err := json.Unmarshal(msg.Payload, &responsePayload); err != nil {
			return fmt.Errorf("failed to unmarshal CheckTagAvailabilityResponse: %w", err)
		}
		responseChan <- responsePayload
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("failed to subscribe to response topic: %w", err)
	}

	// Publish the request to the LeaderboardStreamName
	if err := s.eventBus.Publish(ctxWithTimeout, userevents.LeaderboardStreamName, requestMsg); err != nil {
		return false, fmt.Errorf("failed to publish CheckTagAvailabilityRequest: %w", err)
	}

	// Wait for the response or timeout
	select {
	case <-ctxWithTimeout.Done():
		return false, fmt.Errorf("timeout waiting for tag availability response")
	case responsePayload := <-responseChan:
		return responsePayload.IsAvailable, nil
	}
}

// publishTagAssigned publishes an event that assigns the tag to the leaderboard.
func (s *UserServiceImpl) publishTagAssigned(ctx context.Context, discordID usertypes.DiscordID, tagNumber int) error {
	s.logger.Info("publishTagAssigned",
		slog.String("discord_id", string(discordID)),
		slog.Int("tag_number", tagNumber),
	)

	msgPayload := userevents.TagAssignedRequestPayload{
		DiscordID: discordID,
		TagNumber: tagNumber,
	}
	payloadBytes, err := json.Marshal(msgPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal TagAssignedRequest payload: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.SetContext(ctx)
	msg.Metadata.Set("subject", userevents.TagAssignedRequest)

	if err := s.eventBus.Publish(ctx, userevents.LeaderboardStreamName, msg); err != nil {
		return fmt.Errorf("failed to publish TagAssigned event: %w", err)
	}

	return nil
}
