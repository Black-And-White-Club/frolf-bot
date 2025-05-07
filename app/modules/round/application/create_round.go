package roundservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"github.com/google/uuid"
)

// ValidateAndProcessRound transforms validated round data to an entity
func (s *RoundService) ValidateAndProcessRound(ctx context.Context, payload roundevents.CreateRoundRequestedPayload, timeParser roundtime.TimeParserInterface) (RoundOperationResult, error) {
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
				Failure: roundevents.RoundValidationFailedPayload{
					UserID:       payload.UserID,
					ErrorMessage: errs,
				},
			}, fmt.Errorf("validation failed: %v", errs)
		} else {
			s.metrics.RecordValidationSuccess(ctx)
		}

		// Parse StartTime
		parsedTimeUnix, err := timeParser.ParseUserTimeInput(
			payload.StartTime,
			payload.Timezone,
			roundutil.RealClock{},
		)
		if err != nil {
			s.metrics.RecordTimeParsingError(ctx)
			return RoundOperationResult{
				Failure: roundevents.RoundValidationFailedPayload{
					UserID:       payload.UserID,
					ErrorMessage: []string{err.Error()},
				},
			}, fmt.Errorf("time parsing failed: %w", err)
		} else {
			s.metrics.RecordTimeParsingSuccess(ctx)
		}

		// Check if start time is in the past
		parsedTime := time.Unix(parsedTimeUnix, 0).UTC()
		if parsedTime.Before(time.Now().UTC()) {
			s.metrics.RecordValidationError(ctx)
			return RoundOperationResult{
				Failure: roundevents.RoundValidationFailedPayload{
					UserID:       payload.UserID,
					ErrorMessage: []string{"start time is in the past"},
				},
			}, fmt.Errorf("validation failed: [start time is in the past]")
		}

		// Create round object
		roundObject := roundtypes.Round{
			Title:        roundtypes.Title(payload.Title),
			Description:  &payload.Description,
			Location:     &payload.Location,
			StartTime:    (*sharedtypes.StartTime)(&parsedTime),
			CreatedBy:    payload.UserID,
			State:        roundtypes.RoundStateUpcoming,
			Participants: []roundtypes.Participant{},
		}

		// Create event payload
		createdPayload := roundevents.RoundEntityCreatedPayload{
			Round:            roundObject,
			DiscordChannelID: payload.ChannelID,
			DiscordGuildID:   "",
		}

		// Log round creation
		s.logger.InfoContext(ctx, "Round object created",
			attr.String("title", string(roundObject.Title)),
			attr.String("description", string(*roundObject.Description)),
			attr.String("location", string(*roundObject.Location)),
			attr.Time("start_time", time.Time(*roundObject.StartTime)),
			attr.String("created_by", string(roundObject.CreatedBy)),
		)

		return RoundOperationResult{Success: createdPayload}, nil
	})

	return result, err
}

// StoreRound stores a round in the database
func (s *RoundService) StoreRound(ctx context.Context, payload roundevents.RoundEntityCreatedPayload) (RoundOperationResult, error) {
	result, err := s.serviceWrapper(ctx, "StoreRound", payload.Round.ID, func(ctx context.Context) (RoundOperationResult, error) {
		// Validate round data
		if payload.Round.Title == "" || payload.Round.Description == nil || payload.Round.Location == nil || payload.Round.StartTime == nil {
			s.metrics.RecordValidationError(ctx)
			return RoundOperationResult{
				Failure: roundevents.RoundCreationFailedPayload{
					UserID:       payload.Round.CreatedBy,
					ErrorMessage: "invalid round data",
				},
			}, fmt.Errorf("invalid round data")
		}

		roundTypes := payload.Round
		location := ""
		if roundTypes.Location != nil {
			location = string(*roundTypes.Location)
		}

		defaultType := roundtypes.DefaultEventType
		roundTypes.EventType = &defaultType

		// Map round data to the database model
		roundDB := roundtypes.Round{
			Title:        roundTypes.Title,
			Description:  roundTypes.Description,
			Location:     roundTypes.Location,
			EventType:    roundTypes.EventType,
			StartTime:    payload.Round.StartTime,
			Finalized:    roundTypes.Finalized,
			CreatedBy:    roundTypes.CreatedBy,
			State:        roundTypes.State,
			Participants: []roundtypes.Participant{},
		}

		if roundDB.Description == nil || roundDB.Location == nil || roundDB.StartTime == nil {
			return RoundOperationResult{
				Failure: roundevents.RoundCreationFailedPayload{
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

		s.logger.Debug("🔍 roundDB.Title", attr.String("title", string(roundDB.Title)))

		if roundDB.Description == nil {
			s.logger.Debug("⚠️ Description is nil")
		} else {
			s.logger.Debug("✅ Description exists", attr.String("description", string(*roundDB.Description)))
		}

		if roundDB.Location == nil {
			s.logger.Debug("⚠️ Location is nil")
		} else {
			s.logger.Debug("✅ Location exists", attr.String("location", string(*roundDB.Location)))
		}

		if roundDB.StartTime == nil {
			s.logger.Debug("⚠️ StartTime is nil")
		} else {
			s.logger.Debug("✅ StartTime exists", attr.Time("start_time", time.Time(*roundDB.StartTime)))
		}

		if roundDB.EventType == nil {
			s.logger.Debug("⚠️ EventType is nil")
		} else {
			s.logger.Debug("✅ EventType exists", attr.String("event_type", string(*roundDB.EventType)))
		}

		s.logger.InfoContext(ctx, "About to create round in DB",
			attr.String("title", string(roundDB.Title)),
			attr.String("description", desc),
			attr.String("location", loc),
			attr.Time("start_time", startTime),
			attr.String("created_by", string(roundDB.CreatedBy)),
		)

		// Store the round in the database
		if err := s.RoundDB.CreateRound(ctx, &roundDB); err != nil {
			s.metrics.RecordDBOperationError(ctx, "create_round")
			return RoundOperationResult{
				Failure: roundevents.RoundCreationFailedPayload{
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

		return RoundOperationResult{Success: roundevents.RoundCreatedPayload{
			BaseRoundPayload: roundtypes.BaseRoundPayload{
				RoundID:     roundDB.ID,
				Title:       roundDB.Title,
				Description: roundDB.Description,
				Location:    roundDB.Location,
				StartTime:   roundDB.StartTime,
				UserID:      roundDB.CreatedBy,
			},
			ChannelID: payload.DiscordChannelID,
		}}, nil
	})

	// Return the result and error as-is
	return result, err
}

// UpdateRoundMessageID updates the Discord event message ID for a round in the database
// and returns the updated Round object.
func (s *RoundService) UpdateRoundMessageID(ctx context.Context, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error) {
	result, err := s.serviceWrapper(ctx, "UpdateRoundMessageID", roundID, func(ctx context.Context) (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Attempting to update Discord message ID for round",
			attr.RoundID("round_id", roundID),
			attr.String("discord_message_id", discordMessageID),
		)

		round, dbErr := s.RoundDB.UpdateEventMessageID(ctx, roundID, discordMessageID)
		if dbErr != nil {
			s.metrics.RecordDBOperationError(ctx, "update_round_message_id")
			s.logger.ErrorContext(ctx, "Failed to update Discord event message ID in DB",
				attr.RoundID("round_id", roundID),
				attr.String("discord_message_id", discordMessageID),
				attr.Error(dbErr),
			)
			return RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: roundID,
					Error:   fmt.Sprintf("database update failed: %v", dbErr),
				},
			}, fmt.Errorf("failed to update Discord event message ID in DB: %w", dbErr)
		}

		s.metrics.RecordDBOperationSuccess(ctx, "update_round_message_id")
		s.logger.InfoContext(ctx, "Successfully updated Discord message ID in DB",
			attr.RoundID("round_id", roundID),
			attr.String("discord_message_id", discordMessageID),
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
