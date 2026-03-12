package roundhandlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	clubevents "github.com/Black-And-White-Club/frolf-bot-shared/events/club"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	clubdb "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/repositories"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	"github.com/google/uuid"
)

// HandleCreateRoundRequest handles the initial request to create a round.
func (h *RoundHandlers) HandleCreateRoundRequest(
	ctx context.Context,
	payload *roundevents.CreateRoundRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	canonicalGuildID := payload.GuildID
	var challengeContext *createRoundChallengeContext
	if payload.ChallengeID != nil && strings.TrimSpace(*payload.ChallengeID) != "" {
		var err error
		challengeContext, err = h.challengeContextForCreateRound(ctx, payload.UserID, strings.TrimSpace(*payload.ChallengeID))
		if err != nil {
			var validationErr challengeRoundValidationError
			if errors.As(err, &validationErr) {
				return []handlerwrapper.Result{{
					Topic: roundevents.RoundValidationFailedV1,
					Payload: &roundevents.RoundValidationFailedPayloadV1{
						GuildID:       payload.GuildID,
						UserID:        payload.UserID,
						ErrorMessages: []string{validationErr.Error()},
					},
				}}, nil
			}
			return nil, err
		}
		canonicalGuildID = challengeContext.guildID
	}

	if h.logger != nil {
		h.logger.InfoContext(ctx, "processing create round request",
			attr.ExtractCorrelationID(ctx),
			attr.String("guild_id", string(canonicalGuildID)),
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
		GuildID:     canonicalGuildID,
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
				attr.String("guild_id", string(canonicalGuildID)),
				attr.UserID(payload.UserID),
				attr.Error(err),
			)
		}
		return nil, err
	}

	// Explicitly map results to event payloads
	mappedResult := result.Map(
		func(res *roundtypes.CreateRoundResult) any {
			roundPayload := *res.Round
			roundPayload.GuildID = canonicalGuildID
			if challengeContext != nil {
				roundPayload.Participants = append([]roundtypes.Participant(nil), challengeContext.participants...)
			}

			return &roundevents.RoundEntityCreatedPayloadV1{
				GuildID:          canonicalGuildID,
				Round:            roundPayload,
				DiscordChannelID: res.ChannelID,
				DiscordGuildID:   string(canonicalGuildID),
				Config:           sharedevents.NewGuildConfigFragment(res.GuildConfig),
				RequestSource:    payload.RequestSource,
				ChallengeID:      payload.ChallengeID,
			}
		},
		func(err error) any {
			return &roundevents.RoundValidationFailedPayloadV1{
				GuildID:       canonicalGuildID,
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
				attr.String("guild_id", string(canonicalGuildID)),
				attr.UserID(payload.UserID),
				attr.Any("failure", *mappedResult.Failure),
			)
		} else if mappedResult.Success != nil {
			if successPayload, ok := (*mappedResult.Success).(*roundevents.RoundEntityCreatedPayloadV1); ok {
				h.logger.InfoContext(ctx, "create round request validated successfully",
					attr.ExtractCorrelationID(ctx),
					attr.String("guild_id", string(canonicalGuildID)),
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
				ChannelID:   channelID,
				Config:      sharedevents.NewGuildConfigFragment(res.GuildConfig),
				ChallengeID: payload.ChallengeID,
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

	// PWA-originated creation does not emit a later RoundEventMessageIDUpdate.
	// Trigger scheduling directly so queue-based start works without depending on
	// Discord channel/message metadata.
	if usesDirectRoundScheduling(payload.RequestSource) && mappedResult.Success != nil {
		if createdPayload, ok := (*mappedResult.Success).(*roundevents.RoundCreatedPayloadV1); ok {
			scheduleTitle := createdPayload.Title
			if scheduleTitle == "" {
				scheduleTitle = payload.Round.Title
			}

			scheduleDescription := createdPayload.Description
			if scheduleDescription == "" {
				scheduleDescription = payload.Round.Description
			}

			scheduleLocation := createdPayload.Location
			if scheduleLocation == "" {
				scheduleLocation = payload.Round.Location
			}

			scheduleStartTime := createdPayload.StartTime
			if scheduleStartTime == nil {
				scheduleStartTime = payload.Round.StartTime
			}
			if scheduleStartTime == nil {
				return nil, errors.New("round start time missing from created payload")
			}

			scheduleUserID := createdPayload.UserID
			if scheduleUserID == "" {
				scheduleUserID = payload.Round.CreatedBy
			}

			scheduleConfig := createdPayload.Config
			if scheduleConfig == nil {
				scheduleConfig = payload.Config
			}

			nativeEventPlanned := false
			scheduleResult, scheduleErr := h.service.ScheduleRoundEvents(ctx, &roundtypes.ScheduleRoundEventsRequest{
				GuildID:            createdPayload.GuildID,
				RoundID:            createdPayload.RoundID,
				Title:              scheduleTitle.String(),
				Description:        scheduleDescription.String(),
				Location:           scheduleLocation.String(),
				StartTime:          *scheduleStartTime,
				UserID:             scheduleUserID,
				EventMessageID:     payload.Round.EventMessageID,
				ChannelID:          createdPayload.ChannelID,
				Config:             guildConfigFromFragment(scheduleConfig),
				NativeEventPlanned: &nativeEventPlanned,
			})
			if scheduleErr != nil {
				return nil, scheduleErr
			}
			if scheduleResult.Failure != nil && h.logger != nil {
				h.logger.WarnContext(ctx, "direct round scheduling failed in service",
					attr.ExtractCorrelationID(ctx),
					attr.String("guild_id", string(createdPayload.GuildID)),
					attr.RoundID("round_id", createdPayload.RoundID),
					attr.Any("failure", *scheduleResult.Failure),
				)
			}
		}
	}

	if mappedResult.Success != nil {
		if createdPayload, ok := (*mappedResult.Success).(*roundevents.RoundCreatedPayloadV1); ok {
			if challengeLinkResult := newChallengeRoundLinkResult(createdPayload); challengeLinkResult != nil {
				handlerResults = append(handlerResults, *challengeLinkResult)
			}
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
		EventMessageID:     discordMessageID,
		NativeEventPlanned: payload.NativeEventPlanned,
		ChallengeID:        payload.ChallengeID,
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

func usesDirectRoundScheduling(requestSource *string) bool {
	if requestSource == nil {
		return false
	}

	return strings.EqualFold(strings.TrimSpace(*requestSource), requestSourcePWA)
}

type challengeRoundValidationError struct {
	message string
}

func (e challengeRoundValidationError) Error() string {
	return e.message
}

type createRoundChallengeContext struct {
	guildID      sharedtypes.GuildID
	participants []roundtypes.Participant
}

func (h *RoundHandlers) challengeContextForCreateRound(
	ctx context.Context,
	actorExternalID sharedtypes.DiscordID,
	challengeID string,
) (*createRoundChallengeContext, error) {
	if h.challengeLookup == nil {
		return nil, fmt.Errorf("challenge lookup unavailable")
	}
	if h.userService == nil {
		return nil, fmt.Errorf("user service unavailable")
	}

	parsedChallengeID, err := uuid.Parse(challengeID)
	if err != nil {
		return nil, challengeRoundValidationError{message: "challenge_id must be a valid UUID"}
	}

	challenge, err := h.challengeLookup.GetChallengeByUUID(ctx, nil, parsedChallengeID)
	if err != nil {
		return nil, fmt.Errorf("load challenge for round creation: %w", err)
	}
	if challenge.Status != clubtypes.ChallengeStatusAccepted {
		return nil, challengeRoundValidationError{message: "only accepted challenges can create linked rounds"}
	}

	club, err := h.challengeLookup.GetByUUID(ctx, nil, challenge.ClubUUID)
	if err != nil {
		return nil, fmt.Errorf("load challenge club for round creation: %w", err)
	}
	if club.DiscordGuildID == nil || strings.TrimSpace(*club.DiscordGuildID) == "" {
		return nil, challengeRoundValidationError{message: "challenge club is missing a Discord guild ID"}
	}

	if err := h.validateChallengeRoundActor(ctx, sharedtypes.GuildID(strings.TrimSpace(*club.DiscordGuildID)), actorExternalID, challenge); err != nil {
		return nil, err
	}

	challengerID, err := h.userService.GetDiscordIDByUUID(ctx, challenge.ChallengerUserUUID)
	if err != nil {
		return nil, fmt.Errorf("resolve challenger Discord ID: %w", err)
	}
	if challengerID == "" {
		return nil, challengeRoundValidationError{message: "challenge participants must have linked Discord identities"}
	}

	defenderID, err := h.userService.GetDiscordIDByUUID(ctx, challenge.DefenderUserUUID)
	if err != nil {
		return nil, fmt.Errorf("resolve defender Discord ID: %w", err)
	}
	if defenderID == "" {
		return nil, challengeRoundValidationError{message: "challenge participants must have linked Discord identities"}
	}

	return &createRoundChallengeContext{
		guildID: sharedtypes.GuildID(strings.TrimSpace(*club.DiscordGuildID)),
		participants: []roundtypes.Participant{
			{UserID: challengerID, Response: roundtypes.ResponseAccept},
			{UserID: defenderID, Response: roundtypes.ResponseAccept},
		},
	}, nil
}

func (h *RoundHandlers) validateChallengeRoundActor(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	actorExternalID sharedtypes.DiscordID,
	challenge *clubdb.ClubChallenge,
) error {
	if challenge == nil {
		return fmt.Errorf("challenge is required")
	}
	if actorExternalID == "" {
		return challengeRoundValidationError{message: "challenge round creation requires an actor identity"}
	}

	actorUserUUID, err := h.userService.GetUUIDByDiscordID(ctx, actorExternalID)
	if err == nil && actorUserUUID != uuid.Nil {
		if actorUserUUID == challenge.ChallengerUserUUID || actorUserUUID == challenge.DefenderUserUUID {
			return nil
		}
	}

	roleResult, err := h.userService.GetUserRole(ctx, guildID, actorExternalID)
	if err != nil {
		return fmt.Errorf("resolve challenge scheduling role: %w", err)
	}
	if roleResult.Failure != nil {
		return challengeRoundValidationError{message: "only challenge participants or club staff can schedule a challenge round"}
	}
	if roleResult.Success == nil {
		return challengeRoundValidationError{message: "only challenge participants or club staff can schedule a challenge round"}
	}

	role := *roleResult.Success
	if role != sharedtypes.UserRoleAdmin && role != sharedtypes.UserRoleEditor {
		return challengeRoundValidationError{message: "only challenge participants or club staff can schedule a challenge round"}
	}

	return nil
}

func newChallengeRoundLinkResult(payload *roundevents.RoundCreatedPayloadV1) *handlerwrapper.Result {
	if payload == nil || payload.ChallengeID == nil || *payload.ChallengeID == "" || payload.UserID == "" {
		return nil
	}

	return &handlerwrapper.Result{
		Topic: clubevents.ChallengeRoundLinkRequestedV1,
		Payload: &clubevents.ChallengeRoundLinkRequestedPayloadV1{
			GuildID:         string(payload.GuildID),
			ActorExternalID: string(payload.UserID),
			ChallengeID:     *payload.ChallengeID,
			RoundID:         payload.RoundID.String(),
		},
		Metadata: map[string]string{
			"idempotency_key": fmt.Sprintf("challenge-round-link:%s:%s", *payload.ChallengeID, payload.RoundID.String()),
		},
	}
}
