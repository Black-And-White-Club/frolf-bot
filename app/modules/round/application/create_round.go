package roundservice

import (
	"context"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
)

// ValidateAndProcessRound transforms validated round data to an entity
func (s *RoundService) ValidateAndProcessRound(ctx context.Context, payload roundevents.CreateRoundRequestedPayload, timeParser roundtime.TimeParserInterface) (RoundOperationResult, error) {
	result, err := s.serviceWrapper(ctx, "ValidateAndProcessRound", func() (RoundOperationResult, error) {
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
			s.metrics.RecordValidationError()
			return RoundOperationResult{
				Failure: roundevents.RoundValidationFailedPayload{
					UserID:       payload.UserID,
					ErrorMessage: errs,
				},
			}, fmt.Errorf("validation failed: %v", errs)
		} else {
			s.metrics.RecordValidationSuccess()
		}

		// Parse StartTime
		parsedTimeUnix, err := timeParser.ParseUserTimeInput(
			payload.StartTime,
			payload.Timezone,
			roundutil.RealClock{},
		)
		if err != nil {
			s.metrics.RecordTimeParsingError()
			return RoundOperationResult{
				Failure: roundevents.RoundValidationFailedPayload{
					UserID:       payload.UserID,
					ErrorMessage: []string{err.Error()},
				},
			}, fmt.Errorf("time parsing failed: %w", err)
		} else {
			s.metrics.RecordTimeParsingSuccess()
		}

		// Check if start time is in the past
		parsedTime := time.Unix(parsedTimeUnix, 0).UTC()
		if parsedTime.Before(time.Now().UTC()) {
			s.metrics.RecordValidationError()
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
		s.logger.Info("Round object created",
			attr.String("title", string(roundObject.Title)),
			attr.String("description", string(*roundObject.Description)),
			attr.String("location", string(*roundObject.Location)),
			attr.Time("start_time", time.Time(*roundObject.StartTime)),
			attr.String("created_by", string(roundObject.CreatedBy)),
		)

		return RoundOperationResult{Success: createdPayload}, nil
	})

	// We'll just return the result as is along with the error
	// This allows the caller to handle both the result and error appropriately
	return result, err
}

// StoreRound stores a round in the database
func (s *RoundService) StoreRound(ctx context.Context, payload roundevents.RoundEntityCreatedPayload) (RoundOperationResult, error) {
	result, err := s.serviceWrapper(ctx, "StoreRound", func() (RoundOperationResult, error) {
		// Validate round data
		if payload.Round.Title == "" || payload.Round.Description == nil || payload.Round.Location == nil || payload.Round.StartTime == nil {
			s.metrics.RecordValidationError()
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

		// Map round data to the database model
		roundDB := roundtypes.Round{
			Title:       roundTypes.Title,
			Description: roundTypes.Description,
			Location:    roundTypes.Location,
			EventType:   roundTypes.EventType,
			StartTime:   payload.Round.StartTime,
			Finalized:   roundTypes.Finalized,
			CreatedBy:   roundTypes.CreatedBy,
			State:       roundTypes.State,
		}

		// Log before storing
		s.logger.Info("About to create round in DB",
			attr.String("title", string(roundDB.Title)),
			attr.String("description", string(*roundDB.Description)),
			attr.String("location", string(*roundDB.Location)),
			attr.Time("start_time", time.Time(*roundDB.StartTime)),
			attr.String("created_by", string(roundDB.CreatedBy)),
		)

		// Store the round in the database
		if err := s.RoundDB.CreateRound(ctx, &roundDB); err != nil {
			s.metrics.RecordDBOperationError("create_round")
			return RoundOperationResult{
				Failure: roundevents.RoundCreationFailedPayload{
					UserID:       roundTypes.CreatedBy,
					ErrorMessage: fmt.Sprintf("failed to store round: %v", err),
				},
			}, fmt.Errorf("failed to store round: %w", err)
		} else {
			s.metrics.RecordDBOperationSuccess("create_round")
		}

		// Record successful round creation
		s.metrics.RecordRoundCreated(location)

		// Log after storing
		s.logger.Info("Round created successfully",
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
