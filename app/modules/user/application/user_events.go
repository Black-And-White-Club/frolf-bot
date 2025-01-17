package userservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (s *UserServiceImpl) publishUserRoleUpdated(ctx context.Context, evt events.UserRoleUpdatedPayload) error {
	evtData, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal UserRoleUpdatedPayload: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), evtData)
	msg.SetContext(ctx)
	msg.Metadata.Set("subject", events.UserRoleUpdated)

	// Publish to the UserStreamName
	if err := s.eventBus.Publish(ctx, events.UserStreamName, msg); err != nil {
		return fmt.Errorf("eventBus.Publish UserRoleUpdated: %w", err)
	}

	s.logger.Info("Published UserRoleUpdated event", slog.String("discord_id", string(evt.DiscordID)), slog.String("new_role", evt.NewRole))

	return nil
}
