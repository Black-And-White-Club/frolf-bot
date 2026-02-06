package clubhandlers

import (
	"context"
	"errors"
	"log/slog"

	clubevents "github.com/Black-And-White-Club/frolf-bot-shared/events/club"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	clubservice "github.com/Black-And-White-Club/frolf-bot/app/modules/club/application"
	clubdb "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/repositories"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

// ClubHandlers implements the Handlers interface.
type ClubHandlers struct {
	service clubservice.Service
	logger  *slog.Logger
	tracer  trace.Tracer
}

// NewClubHandlers creates a new ClubHandlers instance.
func NewClubHandlers(
	service clubservice.Service,
	logger *slog.Logger,
	tracer trace.Tracer,
) Handlers {
	return &ClubHandlers{
		service: service,
		logger:  logger,
		tracer:  tracer,
	}
}

// HandleClubInfoRequest handles requests for club info.
func (h *ClubHandlers) HandleClubInfoRequest(ctx context.Context, payload *clubevents.ClubInfoRequestPayloadV1) ([]handlerwrapper.Result, error) {
	ctx, span := h.tracer.Start(ctx, "ClubHandlers.HandleClubInfoRequest")
	defer span.End()

	// Debug: log that we received the request
	h.logger.InfoContext(ctx, "Club info request received",
		slog.String("club_uuid", payload.ClubUUID),
	)

	// Debug: log the reply_to from context
	if rt, ok := ctx.Value(handlerwrapper.CtxKeyReplyTo).(string); ok {
		h.logger.InfoContext(ctx, "Reply-to found in context",
			slog.String("reply_to", rt),
		)
	} else {
		h.logger.WarnContext(ctx, "No reply_to in context")
	}

	clubUUID, err := uuid.Parse(payload.ClubUUID)
	if err != nil {
		h.logger.WarnContext(ctx, "Invalid club UUID in request",
			slog.String("club_uuid", payload.ClubUUID),
			slog.String("error", err.Error()),
		)
		return nil, nil // Invalid request, no response
	}

	info, err := h.service.GetClub(ctx, clubUUID)
	if err != nil {
		if errors.Is(err, clubdb.ErrNotFound) {
			h.logger.WarnContext(ctx, "Club not found",
				slog.String("club_uuid", payload.ClubUUID),
			)
			// Return successful response with null/empty fields to prevent client timeout
			// This allows the PWA to see "Name Missing" instead of indefinite loading
			return []handlerwrapper.Result{{
				Topic: clubevents.ClubInfoResponseV1, // Will be overridden by dynamic reply logic below if needed
				Payload: &clubevents.ClubInfoResponsePayloadV1{
					UUID: payload.ClubUUID,
					Name: "Club Not Found",
				},
			}}, nil
		}
		return nil, err
	}

	// Determine reply topic (dynamic ReplyTo takes precedence over static constant)
	replyTopic := clubevents.ClubInfoResponseV1
	if rt, ok := ctx.Value(handlerwrapper.CtxKeyReplyTo).(string); ok && rt != "" {
		replyTopic = rt
	}

	return []handlerwrapper.Result{{
		Topic: replyTopic,
		Payload: &clubevents.ClubInfoResponsePayloadV1{
			UUID:    info.UUID,
			Name:    info.Name,
			IconURL: info.IconURL,
		},
	}}, nil
}

// HandleGuildSetup handles guild setup events to sync club info.
// After upserting the club, it forwards the payload to trigger guild config creation.
func (h *ClubHandlers) HandleGuildSetup(ctx context.Context, payload *guildevents.GuildSetupPayloadV1) ([]handlerwrapper.Result, error) {
	ctx, span := h.tracer.Start(ctx, "ClubHandlers.HandleGuildSetup")
	defer span.End()

	h.logger.InfoContext(ctx, "Club handler received guild setup event",
		slog.String("guild_id", string(payload.GuildID)),
		slog.String("guild_name", payload.GuildName),
	)

	// Use UpsertClubFromDiscord to create or update club
	clubInfo, err := h.service.UpsertClubFromDiscord(ctx, string(payload.GuildID), payload.GuildName, payload.IconURL)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to upsert club from guild setup",
			slog.String("guild_id", string(payload.GuildID)),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	h.logger.InfoContext(ctx, "Club upserted successfully from guild setup",
		slog.String("guild_id", string(payload.GuildID)),
		slog.String("club_uuid", clubInfo.UUID),
		slog.String("club_name", clubInfo.Name),
	)

	// Forward the config fields to trigger guild config creation
	// This ensures guild config is created after the club is successfully upserted
	return []handlerwrapper.Result{{
		Topic: guildevents.GuildConfigCreationRequestedV1,
		Payload: &guildevents.GuildConfigCreationRequestedPayloadV1{
			GuildID:              payload.GuildID,
			SignupChannelID:      payload.SignupChannelID,
			SignupMessageID:      payload.SignupMessageID,
			EventChannelID:       payload.EventChannelID,
			LeaderboardChannelID: payload.LeaderboardChannelID,
			UserRoleID:           payload.UserRoleID,
			EditorRoleID:         payload.EditorRoleID,
			AdminRoleID:          payload.AdminRoleID,
			SignupEmoji:          payload.SignupEmoji,
			AutoSetupCompleted:   payload.AutoSetupCompleted,
			SetupCompletedAt:     payload.SetupCompletedAt,
		},
	}}, nil
}

// HandleClubSyncFromDiscord handles cross-module club sync events published
// by the user module during signup when guild metadata is present.
func (h *ClubHandlers) HandleClubSyncFromDiscord(ctx context.Context, payload *sharedevents.ClubSyncFromDiscordRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload.GuildName == "" {
		return nil, nil
	}

	ctx, span := h.tracer.Start(ctx, "ClubHandlers.HandleClubSyncFromDiscord")
	defer span.End()

	_, err := h.service.UpsertClubFromDiscord(ctx, string(payload.GuildID), payload.GuildName, payload.IconURL)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to upsert club from discord sync",
			slog.String("guild_id", string(payload.GuildID)),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return nil, nil
}
