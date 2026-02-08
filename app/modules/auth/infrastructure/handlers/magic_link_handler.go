package authhandlers

import (
	"context"
	"encoding/json"
	"strings"

	authevents "github.com/Black-And-White-Club/frolf-bot-shared/events/auth"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go"
)

// MagicLinkRequest is the incoming request payload for magic link generation
type MagicLinkRequest struct {
	UserID        string `json:"user_id"`
	GuildID       string `json:"guild_id"`
	Role          string `json:"role"`
	CorrelationID string `json:"correlation_id"`
}

// HandleMagicLinkRequest handles incoming magic link requests via NATS.
func (h *AuthHandlers) HandleMagicLinkRequest(msg *nats.Msg) {
	ctx := context.Background()
	ctx, span := h.tracer.Start(ctx, "AuthHandlers.HandleMagicLinkRequest")
	defer span.End()

	var req MagicLinkRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.logger.ErrorContext(ctx, "Failed to unmarshal magic link request",
			attr.Error(err),
		)
		// Cannot reply if we can't parse the request
		return
	}

	// Sanitize CorrelationID
	if len(req.CorrelationID) > 64 {
		req.CorrelationID = req.CorrelationID[:64]
	}
	// Simple alphanumeric + hyphen check using strings.Builder for efficiency
	var sb strings.Builder
	sb.Grow(len(req.CorrelationID))
	for _, r := range req.CorrelationID {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			sb.WriteRune(r)
		}
	}
	req.CorrelationID = sb.String()

	h.logger.InfoContext(ctx, "Generating magic link",
		attr.String("user_id", req.UserID),
		attr.String("guild_id", req.GuildID),
		attr.String("role", req.Role),
		attr.String("correlation_id", req.CorrelationID),
	)

	// Convert role string to domain type
	role := authdomain.Role(req.Role)

	// Delegate to service
	response, err := h.service.GenerateMagicLink(ctx, req.UserID, req.GuildID, role)

	// Prepare response payload
	respPayload := authevents.MagicLinkGeneratedPayload{
		UserID:        req.UserID,
		GuildID:       req.GuildID,
		CorrelationID: req.CorrelationID,
	}

	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to generate magic link",
			attr.Error(err),
			attr.String("user_id", req.UserID),
		)
		respPayload.Success = false
		respPayload.Error = err.Error()
	} else if !response.Success {
		h.logger.WarnContext(ctx, "Magic link generation returned business error",
			attr.String("error", response.Error),
			attr.String("user_id", req.UserID),
		)
		respPayload.Success = false
		respPayload.Error = response.Error
	} else {
		respPayload.Success = true
		respPayload.URL = response.URL
		h.logger.InfoContext(ctx, "Magic link generated successfully",
			attr.String("user_id", req.UserID),
			attr.String("guild_id", req.GuildID),
		)

		// Handle Profile Sync if needed
		if response.NeedsSync {
			syncPayload := userevents.UserProfileSyncRequestPayloadV1{
				UserID:  sharedtypes.DiscordID(req.UserID),
				GuildID: sharedtypes.GuildID(req.GuildID),
			}

			payloadBytes, _ := json.Marshal(syncPayload) // Ignoring error as struct is simple
			syncMsg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			syncMsg.Metadata.Set("topic", userevents.UserProfileSyncRequestTopicV1)
			syncMsg.Metadata.Set("user_id", req.UserID)
			syncMsg.Metadata.Set("guild_id", req.GuildID)

			if err := h.eventBus.Publish(userevents.UserProfileSyncRequestTopicV1, syncMsg); err != nil {
				h.logger.WarnContext(ctx, "Failed to publish profile sync request",
					attr.Error(err),
				)
			} else {
				h.logger.InfoContext(ctx, "Published profile sync request",
					attr.String("user_id", req.UserID),
				)
			}
		}
	}

	// Publish response event
	outMsg, err := h.helper.CreateNewMessage(respPayload, authevents.MagicLinkGeneratedV1)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to create magic link response message", attr.Error(err))
		return
	}

	// Ensure Nats-Msg-Id is unique (Watermill helper does this, but good to be sure)
	// Publish to the event bus
	if err := h.eventBus.Publish(authevents.MagicLinkGeneratedV1, outMsg); err != nil {
		h.logger.ErrorContext(ctx, "Failed to publish magic link response", attr.Error(err))
		return
	}

	h.logger.InfoContext(ctx, "Published magic link response",
		attr.String("correlation_id", req.CorrelationID),
		attr.Bool("success", respPayload.Success),
	)
}
