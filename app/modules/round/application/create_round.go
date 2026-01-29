package roundservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ValidateRoundCreationWithClock transforms validated round data to an entity
func (s *RoundService) ValidateRoundCreationWithClock(ctx context.Context, req *roundtypes.CreateRoundInput, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (CreateRoundResult, error) {
	result, err := withTelemetry(s, ctx, "ValidateRoundCreation", sharedtypes.RoundID(uuid.Nil), func(ctx context.Context) (CreateRoundResult, error) {
		// Validate the round
		errs := s.roundValidator.ValidateRoundInput(*req)
		if len(errs) > 0 {
			s.metrics.RecordValidationError(ctx)
			s.logger.WarnContext(ctx, "Round validation failed",
				attr.String("user_id", string(req.UserID)),
				attr.Any("validation_errors", errs),
				attr.String("title", string(req.Title)),
			)
			// Return a combined error
			return results.FailureResult[*roundtypes.CreateRoundResult](fmt.Errorf("validation failed: %v", errs)), nil
		} else {
			s.metrics.RecordValidationSuccess(ctx)
		}

		// Parse StartTime
		parsedTimeUnix, err := timeParser.ParseUserTimeInput(
			req.StartTime,
			roundtypes.Timezone(req.Timezone),
			clock,
		)
		if err != nil {
			s.metrics.RecordTimeParsingError(ctx)
			s.logger.WarnContext(ctx, "Time parsing failed",
				attr.String("user_id", string(req.UserID)),
				attr.String("start_time_input", req.StartTime),
				attr.String("timezone", req.Timezone),
				attr.Error(err),
			)
			return results.FailureResult[*roundtypes.CreateRoundResult](fmt.Errorf("time parsing failed: %w", err)), nil
		} else {
			s.metrics.RecordTimeParsingSuccess(ctx)
		}

		// Check if start time is in the past
		// Truncate both times to minute precision to match time parser behavior
		parsedTime := time.Unix(parsedTimeUnix, 0).UTC().Truncate(time.Minute)
		currentTime := time.Now().UTC().Truncate(time.Minute)
		if parsedTime.Before(currentTime) {
			s.metrics.RecordValidationError(ctx)
			s.logger.WarnContext(ctx, "Start time is in the past",
				attr.String("user_id", string(req.UserID)),
				attr.Time("parsed_time", parsedTime),
				attr.Time("current_time", currentTime),
				attr.String("title", string(req.Title)),
			)
			return results.FailureResult[*roundtypes.CreateRoundResult](errors.New("start time is in the past")), nil
		}

		// Create round object
		roundObject := roundtypes.Round{
			ID:           sharedtypes.RoundID(uuid.New()),
			Title:        req.Title,
			Description:  req.Description,
			Location:     req.Location,
			StartTime:    (*sharedtypes.StartTime)(&parsedTime),
			CreatedBy:    req.UserID,
			State:        roundtypes.RoundStateUpcoming,
			Participants: []roundtypes.Participant{},
		}

		// Create result
		createResult := &roundtypes.CreateRoundResult{
			Round:     &roundObject,
			ChannelID: req.ChannelID,
		}

		// Enrich with config fragment if service can supply a guild config (non-fatal if absent)
		if cfg := s.getGuildConfigForEnrichment(ctx, req.GuildID); cfg != nil {
			createResult.GuildConfig = cfg
		}

		// Log round creation
		s.logger.InfoContext(ctx, "Round object created",
			attr.String("title", string(roundObject.Title)),
			attr.String("description", string(roundObject.Description)),
			attr.String("location", string(roundObject.Location)),
			attr.Time("start_time", time.Time(*roundObject.StartTime)),
			attr.String("created_by", string(roundObject.CreatedBy)),
		)

		return results.SuccessResult[*roundtypes.CreateRoundResult, error](createResult), nil
	})

	return result, err
}

// StoreRound stores a round in the database
func (s *RoundService) StoreRound(ctx context.Context, round *roundtypes.Round, guildID sharedtypes.GuildID) (CreateRoundResult, error) {
	storeOp := func(ctx context.Context, db bun.IDB) (CreateRoundResult, error) {
		// Validate round data
		if round.Title == "" || round.Description == "" || round.Location == "" || round.StartTime == nil {
			s.metrics.RecordValidationError(ctx)
			return results.FailureResult[*roundtypes.CreateRoundResult](errors.New("invalid round data")), nil
		}

		defaultType := roundtypes.DefaultEventType
		if round.EventType == nil {
			round.EventType = &defaultType
		}

		startTime := time.Time(*round.StartTime)

		s.logger.InfoContext(ctx, "About to create round in DB",
			attr.String("title", string(round.Title)),
			attr.String("description", string(round.Description)),
			attr.String("location", string(round.Location)),
			attr.Time("start_time", startTime),
			attr.String("created_by", string(round.CreatedBy)),
		)

		// Store the round in the database
		if err := s.repo.CreateRound(ctx, db, guildID, round); err != nil {
			s.metrics.RecordDBOperationError(ctx, "create_round")
			return results.FailureResult[*roundtypes.CreateRoundResult](fmt.Errorf("failed to store round: %w", err)), fmt.Errorf("failed to store round: %w", err)
		} else {
			s.metrics.RecordDBOperationSuccess(ctx, "create_round")
		}
		// Immediately materialize participant groups (singles-safe)
		hasGroups, err := s.repo.RoundHasGroups(ctx, db, round.ID)
		if err != nil {
			return results.FailureResult[*roundtypes.CreateRoundResult](fmt.Errorf("failed checking round groups: %w", err)), err
		}

		if !hasGroups {
			if err := s.repo.CreateRoundGroups(
				ctx,
				db,
				round.ID,
				round.Participants,
			); err != nil {
				return results.FailureResult[*roundtypes.CreateRoundResult](fmt.Errorf("failed creating round groups: %w", err)), err
			}
		}

		// Record successful round creation
		s.metrics.RecordRoundCreated(ctx, string(round.Location))

		// Log after storing
		s.logger.InfoContext(ctx, "Round created successfully",
			attr.StringUUID("round_id", round.ID.String()),
			attr.String("title", string(round.Title)),
			attr.String("description", string(round.Description)),
			attr.String("location", string(round.Location)),
			attr.Time("start_time", time.Time(*round.StartTime)),
			attr.String("created_by", string(round.CreatedBy)),
		)

		created := &roundtypes.CreateRoundResult{
			Round: round,
		}
		if cfg := s.getGuildConfigForEnrichment(ctx, guildID); cfg != nil {
			created.GuildConfig = cfg
		}
		return results.SuccessResult[*roundtypes.CreateRoundResult, error](created), nil
	}

	result, err := withTelemetry(s, ctx, "StoreRound", round.ID, func(ctx context.Context) (CreateRoundResult, error) {
		return runInTx(s, ctx, storeOp)
	})

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

	// Use a short-lived child context to protect DB work from premature cancellation
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := withTelemetry(s, dbCtx, "UpdateRoundMessageID", roundID, func(ctx context.Context) (results.OperationResult[*roundtypes.Round, error], error) {
		s.logger.InfoContext(ctx, "Attempting to update Discord message ID for round",
			attr.RoundID("round_id", roundID),
			attr.String("discord_message_id", discordMessageID),
			attr.Any("guild_id_type", fmt.Sprintf("%T", guildID)),
			attr.Any("round_id_type", fmt.Sprintf("%T", roundID)),
			attr.Any("discord_message_id_type", fmt.Sprintf("%T", discordMessageID)),
			attr.String("guild_id_value", string(guildID)),
			attr.String("round_id_value", roundID.String()),
		)

		round, dbErr := s.repo.UpdateEventMessageID(ctx, s.db, guildID, roundID, discordMessageID)
		if dbErr != nil {
			s.metrics.RecordDBOperationError(ctx, "update_round_message_id")
			s.logger.ErrorContext(ctx, "Failed to update Discord event message ID in DB",
				attr.RoundID("round_id", roundID),
				attr.String("discord_message_id", discordMessageID),
				attr.String("guild_id_value", string(guildID)),
				attr.Error(dbErr),
			)
			return results.FailureResult[*roundtypes.Round](fmt.Errorf("database update failed: %w", dbErr)), fmt.Errorf("failed to update Discord event message ID in DB: %w", dbErr)
		}

		s.metrics.RecordDBOperationSuccess(ctx, "update_round_message_id")
		s.logger.InfoContext(ctx, "Successfully updated Discord message ID in DB",
			attr.RoundID("round_id", roundID),
			attr.String("discord_message_id", discordMessageID),
			attr.String("guild_id_value", string(guildID)),
		)

		return results.SuccessResult[*roundtypes.Round, error](round), nil
	})
	if err != nil {
		return nil, err
	}

	if result.Success != nil {
		return *result.Success, nil
	}

	if result.Failure != nil {
		return nil, fmt.Errorf("operation failed: %w", *result.Failure)
	}
	return nil, errors.New("operation failed with unknown error")
}
