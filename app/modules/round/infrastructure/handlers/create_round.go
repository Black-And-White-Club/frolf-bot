package roundhandlers

import (
	"context"
	"errors"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
)

// HandleCreateRoundRequest handles the initial request to create a round.
func (h *RoundHandlers) HandleCreateRoundRequest(
	ctx context.Context,
	payload *roundevents.CreateRoundRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	if h.logger != nil {
		h.logger.InfoContext(ctx, "processing create round request",
			attr.ExtractCorrelationID(ctx),
			attr.String("guild_id", string(payload.GuildID)),
			attr.UserID(payload.UserID),
			attr.String("title", string(payload.Title)),
			attr.String("start_time", payload.StartTime),
			attr.String("timezone", string(payload.Timezone)),
		)
	}

	clock := h.extractAnchorClock(ctx)

	description := ""
	if payload.Description != nil {
		description = string(*payload.Description)
	}

	descVal := roundtypes.Description(description)
	req := &roundtypes.CreateRoundInput{
		GuildID:     payload.GuildID,
		Title:       roundtypes.Title(payload.Title),
		Description: &descVal,
		Location:    roundtypes.Location(payload.Location),
		StartTime:   payload.StartTime,
		Timezone:    string(payload.Timezone),
		UserID:      payload.UserID,
		ChannelID:   payload.ChannelID,
	}

	result, err := h.service.ValidateRoundCreationWithClock(ctx, req, roundtime.NewTimeParser(), clock)
	if err != nil {
		if h.logger != nil {
			h.logger.ErrorContext(ctx, "create round validation request failed",
				attr.ExtractCorrelationID(ctx),
				attr.String("guild_id", string(payload.GuildID)),
				attr.UserID(payload.UserID),
				attr.Error(err),
			)
		}
		return nil, err
	}

	// Explicitly map results to event payloads
	mappedResult := result.Map(
		func(res *roundtypes.CreateRoundResult) any {
			return &roundevents.RoundEntityCreatedPayloadV1{
				GuildID:          payload.GuildID,
				Round:            *res.Round,
				DiscordChannelID: res.ChannelID,
				DiscordGuildID:   string(payload.GuildID),
				Config:           sharedevents.NewGuildConfigFragment(res.GuildConfig),
			}
		},
		func(err error) any {
			return &roundevents.RoundValidationFailedPayloadV1{
				GuildID:       payload.GuildID,
				UserID:        payload.UserID,
				ErrorMessages: []string{err.Error()},
			}
		},
	)

	handlerResults := mapOperationResult(mappedResult,
		roundevents.RoundEntityCreatedV1,
		roundevents.RoundValidationFailedV1,
	)

	if h.logger != nil {
		if mappedResult.Failure != nil {
			h.logger.WarnContext(ctx, "create round request validation failed",
				attr.ExtractCorrelationID(ctx),
				attr.String("guild_id", string(payload.GuildID)),
				attr.UserID(payload.UserID),
				attr.Any("failure", *mappedResult.Failure),
			)
		} else if mappedResult.Success != nil {
			if successPayload, ok := (*mappedResult.Success).(*roundevents.RoundEntityCreatedPayloadV1); ok {
				h.logger.InfoContext(ctx, "create round request validated successfully",
					attr.ExtractCorrelationID(ctx),
					attr.String("guild_id", string(payload.GuildID)),
					attr.RoundID("round_id", successPayload.Round.ID),
				)
			}
		}
	}

	return handlerResults, nil
}

// HandleRoundEntityCreated handles persisting the round entity to the database.
func (h *RoundHandlers) HandleRoundEntityCreated(
	ctx context.Context,
	payload *roundevents.RoundEntityCreatedPayloadV1,
) ([]handlerwrapper.Result, error) {
	if h.logger != nil {
		h.logger.InfoContext(ctx, "processing round entity created",
			attr.ExtractCorrelationID(ctx),
			attr.String("guild_id", string(payload.GuildID)),
			attr.RoundID("round_id", payload.Round.ID),
		)
	}

	result, err := h.service.StoreRound(ctx, &payload.Round, payload.GuildID)
	if err != nil {
		if h.logger != nil {
			h.logger.ErrorContext(ctx, "store round failed",
				attr.ExtractCorrelationID(ctx),
				attr.String("guild_id", string(payload.GuildID)),
				attr.RoundID("round_id", payload.Round.ID),
				attr.Error(err),
			)
		}
		return nil, err
	}

	// Explicitly map the result to the event payload to ensure correct structure
	mappedResult := result.Map(
		func(res *roundtypes.CreateRoundResult) any {
			r := res.Round
			channelID := payload.DiscordChannelID
			if channelID == "" && res.GuildConfig != nil {
				channelID = res.GuildConfig.EventChannelID
			}

			createdPayload := &roundevents.RoundCreatedPayloadV1{
				GuildID: payload.GuildID,
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID:     r.ID,
					Title:       r.Title,
					Description: r.Description,
					Location:    r.Location,
					StartTime:   r.StartTime,
					UserID:      r.CreatedBy,
				},
				ChannelID: channelID,
				Config:    sharedevents.NewGuildConfigFragment(res.GuildConfig),
			}

			return createdPayload
		},
		func(err error) any {
			channelID := payload.DiscordChannelID
			if channelID == "" && payload.Config != nil {
				channelID = payload.Config.EventChannelID
			}

			return &roundevents.RoundCreationFailedPayloadV1{
				GuildID:      payload.GuildID,
				UserID:       payload.Round.CreatedBy,
				ErrorMessage: err.Error(),
				ChannelID:    channelID,
			}
		},
	)

	handlerResults := mapOperationResult(mappedResult,
		roundevents.RoundCreatedV2,
		roundevents.RoundCreationFailedV1,
	)

	// Add both legacy GuildID and internal ClubUUID scoped versions for PWA/NATS transition
	handlerResults = h.addParallelIdentityResults(ctx, handlerResults, roundevents.RoundCreatedV2, payload.GuildID)

	// PWA-only creation path does not emit a later RoundEventMessageIDUpdate.
	// Trigger scheduling directly so queue-based start works without Discord native events.
	if payload.DiscordChannelID == "" && mappedResult.Success != nil {
		if createdPayload, ok := (*mappedResult.Success).(*roundevents.RoundCreatedPayloadV1); ok {
			handlerResults = append(handlerResults, handlerwrapper.Result{
				Topic: roundevents.RoundEventMessageIDUpdatedV1,
				Payload: &roundevents.RoundScheduledPayloadV1{
					GuildID: createdPayload.GuildID,
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     createdPayload.RoundID,
						Title:       createdPayload.Title,
						Description: createdPayload.Description,
						Location:    createdPayload.Location,
						StartTime:   createdPayload.StartTime,
						UserID:      createdPayload.UserID,
					},
					EventMessageID: payload.Round.EventMessageID,
					Config:         createdPayload.Config,
					ChannelID:      createdPayload.ChannelID,
				},
			})
		}
	}

	if h.logger != nil {
		if mappedResult.Failure != nil {
			h.logger.WarnContext(ctx, "round creation persistence failed",
				attr.ExtractCorrelationID(ctx),
				attr.String("guild_id", string(payload.GuildID)),
				attr.RoundID("round_id", payload.Round.ID),
				attr.Any("failure", *mappedResult.Failure),
			)
		} else if mappedResult.Success != nil {
			if successPayload, ok := (*mappedResult.Success).(*roundevents.RoundCreatedPayloadV1); ok {
				h.logger.InfoContext(ctx, "round creation persisted successfully",
					attr.ExtractCorrelationID(ctx),
					attr.String("guild_id", string(successPayload.GuildID)),
					attr.RoundID("round_id", successPayload.RoundID),
					attr.Int("published_events", len(handlerResults)),
				)
			}
		}
	}

	return handlerResults, nil
}

// HandleRoundEventMessageIDUpdate updates the round with the Discord message ID.
// Compatibility with the Discord module on Main.
func (h *RoundHandlers) HandleRoundEventMessageIDUpdate(
	ctx context.Context,
	payload *roundevents.RoundMessageIDUpdatePayloadV1,
) ([]handlerwrapper.Result, error) {
	if h.logger != nil {
		h.logger.InfoContext(ctx, "processing round event message id update",
			attr.ExtractCorrelationID(ctx),
			attr.String("guild_id", string(payload.GuildID)),
			attr.RoundID("round_id", payload.RoundID),
		)
	}

	// 1. Extract metadata injected into context by the wrapper
	discordMessageID, ok := ctx.Value("discord_message_id").(string)
	if !ok || discordMessageID == "" {
		return nil, errors.New("discord_message_id missing from context")
	}

	// 2. Call service to persist the ID
	updatedRound, err := h.service.UpdateRoundMessageID(ctx, payload.GuildID, payload.RoundID, discordMessageID)
	if err != nil {
		if h.logger != nil {
			h.logger.ErrorContext(ctx, "round event message id update failed",
				attr.ExtractCorrelationID(ctx),
				attr.String("guild_id", string(payload.GuildID)),
				attr.RoundID("round_id", payload.RoundID),
				attr.Error(err),
			)
		}
		return nil, err
	}

	if updatedRound == nil {
		return nil, errors.New("updated round object is nil")
	}

	// 3. Construct outgoing payload
	scheduledPayload := roundevents.RoundScheduledPayloadV1{
		GuildID: payload.GuildID,
		BaseRoundPayload: roundtypes.BaseRoundPayload{
			RoundID:     updatedRound.ID,
			Title:       updatedRound.Title,
			Description: updatedRound.Description,
			Location:    updatedRound.Location,
			StartTime:   updatedRound.StartTime,
			UserID:      updatedRound.CreatedBy,
		},
		EventMessageID: discordMessageID,
	}

	if h.logger != nil {
		h.logger.InfoContext(ctx, "round event message id update persisted",
			attr.ExtractCorrelationID(ctx),
			attr.String("guild_id", string(payload.GuildID)),
			attr.RoundID("round_id", payload.RoundID),
			attr.String("discord_message_id", discordMessageID),
		)
	}

	// 4. Explicitly promote the metadata to the outgoing message headers.
	// Without this, the Discord Module will not know which message to track.
	return []handlerwrapper.Result{
		{
			Topic:   roundevents.RoundEventMessageIDUpdatedV1,
			Payload: scheduledPayload,
			Metadata: map[string]string{
				"discord_message_id": discordMessageID,
			},
		},
	}, nil
}
