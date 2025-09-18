package roundservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"github.com/google/uuid"
)

// ValidateAndProcessRound transforms validated round data to an entity
// ValidateAndProcessRoundWithClock is the internal implementation allowing a custom clock (e.g. anchored).
func (s *RoundService) ValidateAndProcessRoundWithClock(ctx context.Context, payload roundevents.CreateRoundRequestedPayload, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (RoundOperationResult, error) {
	result, err := s.serviceWrapper(ctx, "ValidateAndProcessRound", sharedtypes.RoundID(uuid.Nil), func(ctx context.Context) (RoundOperationResult, error) {
		// Validate the round
		input := roundtypes.CreateRoundInput{
			Title:       payload.Title,
			Description: &payload.Description,
			Location:    &payload.Location,
			StartTime:   payload.StartTime,
			UserID:      payload.UserID,
		}

		errs := s.roundValidator.ValidateRoundInput(input)
		if len(errs) > 0 {
			s.metrics.RecordValidationError(ctx)
			return RoundOperationResult{
				Failure: &roundevents.RoundValidationFailedPayload{
					UserID:       payload.UserID,
					ErrorMessage: errs,
				},
			}, nil // ← Changed from fmt.Errorf to nil
		} else {
			s.metrics.RecordValidationSuccess(ctx)
		}

		// Parse StartTime
		parsedTimeUnix, err := timeParser.ParseUserTimeInput(
			payload.StartTime,
			payload.Timezone,
			clock,
		)
		if err != nil {
			s.metrics.RecordTimeParsingError(ctx)
			return RoundOperationResult{
				Failure: &roundevents.RoundValidationFailedPayload{
					UserID:       payload.UserID,
					ErrorMessage: []string{err.Error()},
				},
			}, nil // ← Changed from fmt.Errorf to nil
		} else {
			s.metrics.RecordTimeParsingSuccess(ctx)
		}

		// Check if start time is in the past
		parsedTime := time.Unix(parsedTimeUnix, 0).UTC()
		if parsedTime.Before(time.Now().UTC()) {
			s.metrics.RecordValidationError(ctx)
			return RoundOperationResult{
				Failure: &roundevents.RoundValidationFailedPayload{
					UserID:       payload.UserID,
					ErrorMessage: []string{"start time is in the past"},
				},
			}, nil // ← Changed from fmt.Errorf to nil
		}

		// Create round object
		roundObject := roundtypes.Round{
			ID:           sharedtypes.RoundID(uuid.New()),
			Title:        roundtypes.Title(payload.Title),
			Description:  &payload.Description,
			Location:     &payload.Location,
			StartTime:    (*sharedtypes.StartTime)(&parsedTime),
			CreatedBy:    payload.UserID,
			State:        roundtypes.RoundStateUpcoming,
			Participants: []roundtypes.Participant{},
		}

		// Create event payload, propagate GuildID from payload
		createdPayload := roundevents.RoundEntityCreatedPayload{
			GuildID:          payload.GuildID,
			Round:            roundObject,
			DiscordChannelID: payload.ChannelID,
			DiscordGuildID:   string(payload.GuildID), // maintain existing behavior
		}

		// Enrich with config fragment if service can supply a guild config (non-fatal if absent)
		if cfg := s.getGuildConfigForEnrichment(ctx, payload.GuildID); cfg != nil {
			createdPayload.Config = sharedevents.NewGuildConfigFragment(cfg)
		}

		// Log round creation
		s.logger.InfoContext(ctx, "Round object created",
			attr.String("title", string(roundObject.Title)),
			attr.String("description", string(*roundObject.Description)),
			attr.String("location", string(*roundObject.Location)),
			attr.Time("start_time", time.Time(*roundObject.StartTime)),
			attr.String("created_by", string(roundObject.CreatedBy)),
		)

		return RoundOperationResult{Success: &createdPayload}, nil
	})

	return result, err
}

// ValidateAndProcessRound keeps backward compatibility using the real clock.
func (s *RoundService) ValidateAndProcessRound(ctx context.Context, payload roundevents.CreateRoundRequestedPayload, timeParser roundtime.TimeParserInterface) (RoundOperationResult, error) {
	return s.ValidateAndProcessRoundWithClock(ctx, payload, timeParser, roundutil.RealClock{})
}

// StoreRound stores a round in the database
func (s *RoundService) StoreRound(ctx context.Context, guildID sharedtypes.GuildID, payload roundevents.RoundEntityCreatedPayload) (RoundOperationResult, error) {
	result, err := s.serviceWrapper(ctx, "StoreRound", payload.Round.ID, func(ctx context.Context) (RoundOperationResult, error) {
		// Validate round data
		if payload.Round.Title == "" || payload.Round.Description == nil || payload.Round.Location == nil || payload.Round.StartTime == nil {
			s.metrics.RecordValidationError(ctx)
			return RoundOperationResult{
				Failure: &roundevents.RoundCreationFailedPayload{
					UserID:       payload.Round.CreatedBy,
					ErrorMessage: "invalid round data",
				},
			}, fmt.Errorf("invalid round data")
		}

		roundTypes := payload.Round
		// Ensure GuildID is always set on the DB model for multi-tenant safety
		roundTypes.GuildID = guildID
		location := ""
		if roundTypes.Location != nil {
			location = string(*roundTypes.Location)
		}

		defaultType := roundtypes.DefaultEventType
		roundTypes.EventType = &defaultType

		// Map round data to the database model
		roundDB := roundtypes.Round{
			ID:           payload.Round.ID,
			Title:        roundTypes.Title,
			Description:  roundTypes.Description,
			Location:     roundTypes.Location,
			EventType:    roundTypes.EventType,
			StartTime:    payload.Round.StartTime,
			Finalized:    roundTypes.Finalized,
			CreatedBy:    roundTypes.CreatedBy,
			State:        roundTypes.State,
			Participants: []roundtypes.Participant{},
			GuildID:      guildID,
		}

		if roundDB.Description == nil || roundDB.Location == nil || roundDB.StartTime == nil {
			return RoundOperationResult{
				Failure: &roundevents.RoundCreationFailedPayload{
					UserID:       roundDB.CreatedBy,
					ErrorMessage: "one or more required fields are nil",
				},
			}, fmt.Errorf("nil field: desc=%v, loc=%v, start=%v", roundDB.Description, roundDB.Location, roundDB.StartTime)
		}

		// Safely dereference optional fields for logging
		desc := ""
		if roundDB.Description != nil {
			desc = string(*roundDB.Description)
		}

		loc := ""
		if roundDB.Location != nil {
			loc = string(*roundDB.Location)
		}

		startTime := time.Time{}
		if roundDB.StartTime != nil {
			startTime = time.Time(*roundDB.StartTime)
		}

		s.logger.InfoContext(ctx, "About to create round in DB",
			attr.String("title", string(roundDB.Title)),
			attr.String("description", desc),
			attr.String("location", loc),
			attr.Time("start_time", startTime),
			attr.String("created_by", string(roundDB.CreatedBy)),
		)

		// Store the round in the database
		if err := s.RoundDB.CreateRound(ctx, guildID, &roundDB); err != nil {
			s.metrics.RecordDBOperationError(ctx, "create_round")
			return RoundOperationResult{
				Failure: &roundevents.RoundCreationFailedPayload{
					UserID:       roundTypes.CreatedBy,
					ErrorMessage: fmt.Sprintf("failed to store round: %v", err),
				},
			}, fmt.Errorf("failed to store round: %w", err)
		} else {
			s.metrics.RecordDBOperationSuccess(ctx, "create_round")
		}

		// Record successful round creation
		s.metrics.RecordRoundCreated(ctx, location)

		// Log after storing
		s.logger.InfoContext(ctx, "Round created successfully",
			attr.StringUUID("round_id", roundDB.ID.String()),
			attr.String("title", string(roundDB.Title)),
			attr.String("description", string(*roundDB.Description)),
			attr.String("location", string(*roundDB.Location)),
			attr.Time("start_time", time.Time(*roundDB.StartTime)),
			attr.String("created_by", string(roundDB.CreatedBy)),
		)

		created := &roundevents.RoundCreatedPayload{
			BaseRoundPayload: roundtypes.BaseRoundPayload{
				RoundID:     roundDB.ID,
				Title:       roundDB.Title,
				Description: roundDB.Description,
				Location:    roundDB.Location,
				StartTime:   roundDB.StartTime,
				UserID:      roundDB.CreatedBy,
			},
			ChannelID: payload.DiscordChannelID,
		}
		if cfg := s.getGuildConfigForEnrichment(ctx, guildID); cfg != nil {
			created.Config = sharedevents.NewGuildConfigFragment(cfg)
		}
		return RoundOperationResult{Success: created}, nil
	})

	// Return the result and error as-is
	return result, err
}

// UpdateRoundMessageID updates the Discord event message ID for a round in the database
// and returns the updated Round object.
func (s *RoundService) UpdateRoundMessageID(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error) {
	// Note: guildID may be empty in some integration test flows where the test
	// data was inserted without a guild. Log a warning but proceed; the DB
	// layer filters by the provided guildID value.
	if string(guildID) == "" {
		s.logger.WarnContext(ctx, "UpdateRoundMessageID proceeding with empty guildID",
			attr.RoundID("round_id", roundID),
			attr.String("discord_message_id", discordMessageID),
		)
	}

	result, err := s.serviceWrapper(ctx, "UpdateRoundMessageID", roundID, func(ctx context.Context) (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Attempting to update Discord message ID for round",
			attr.RoundID("round_id", roundID),
			attr.String("discord_message_id", discordMessageID),
			attr.Any("guild_id_type", fmt.Sprintf("%T", guildID)),
			attr.Any("round_id_type", fmt.Sprintf("%T", roundID)),
			attr.Any("discord_message_id_type", fmt.Sprintf("%T", discordMessageID)),
			attr.String("guild_id_value", string(guildID)),
			attr.String("round_id_value", roundID.String()),
		)

		round, dbErr := s.RoundDB.UpdateEventMessageID(ctx, guildID, roundID, discordMessageID)
		if dbErr != nil {
			s.metrics.RecordDBOperationError(ctx, "update_round_message_id")
			s.logger.ErrorContext(ctx, "Failed to update Discord event message ID in DB",
				attr.RoundID("round_id", roundID),
				attr.String("discord_message_id", discordMessageID),
				attr.String("guild_id_value", string(guildID)),
				attr.Error(dbErr),
			)
			return RoundOperationResult{
				Failure: &roundevents.RoundErrorPayload{
					RoundID: roundID,
					Error:   fmt.Sprintf("database update failed: %v", dbErr),
				},
			}, fmt.Errorf("failed to update Discord event message ID in DB: %w", dbErr)
		}

		s.metrics.RecordDBOperationSuccess(ctx, "update_round_message_id")
		s.logger.InfoContext(ctx, "Successfully updated Discord message ID in DB",
			attr.RoundID("round_id", roundID),
			attr.String("discord_message_id", discordMessageID),
			attr.String("guild_id_value", string(guildID)),
		)

		return RoundOperationResult{Success: round}, nil
	})
	if err != nil {
		return nil, err
	}

	if result.Success != nil {
		updatedRound, ok := result.Success.(*roundtypes.Round)
		if !ok {
			s.logger.ErrorContext(ctx, "Unexpected success result type from serviceWrapper",
				attr.RoundID("round_id", roundID),
				attr.Any("result_type", fmt.Sprintf("%T", result.Success)),
			)
			return nil, errors.New("internal service error: unexpected result type")
		}
		return updatedRound, nil
	}

	if result.Failure == nil {
		s.logger.ErrorContext(ctx, "Service wrapper returned no error, success, or failure result",
			attr.RoundID("round_id", roundID),
		)
		return nil, errors.New("internal service error: no result received")
	}

	failurePayload, ok := result.Failure.(roundevents.RoundErrorPayload)
	if ok {
		return nil, fmt.Errorf("operation failed: %s", failurePayload.Error)
	}
	return nil, errors.New("operation failed with unknown error")
}
