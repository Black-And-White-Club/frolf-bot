package roundservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"github.com/google/uuid"
)

// ValidateRoundUpdateWithClock validates and processes round update with time parsing (like create round)
func (s *RoundService) ValidateRoundUpdateWithClock(ctx context.Context, req *roundtypes.UpdateRoundRequest, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (UpdateRoundResult, error) {
	return withTelemetry(s, ctx, "ValidateRoundUpdate", req.RoundID, func(ctx context.Context) (UpdateRoundResult, error) {
		s.logger.InfoContext(ctx, "Validating and processing round update request",
			attr.ExtractCorrelationID(ctx),
			attr.RoundID("round_id", req.RoundID),
		)

		var errs []string

		// Basic validation checks (like create round)
		if req.RoundID == sharedtypes.RoundID(uuid.Nil) {
			errs = append(errs, "round ID cannot be zero")
		}

		if req.Title == nil && req.Description == nil && req.Location == nil && req.StartTime == nil {
			errs = append(errs, "at least one field to update must be provided")
		}

		// Process time string if provided (exactly like create round) with nil-safe timezone handling
		var parsedStartTime *sharedtypes.StartTime
		if req.StartTime != nil && *req.StartTime != "" {
			if req.Timezone == nil || *req.Timezone == "" {
				errs = append(errs, "timezone is required when providing start time")
			} else {
				s.logger.InfoContext(ctx, "Processing time string for round update",
					attr.ExtractCorrelationID(ctx),
					attr.RoundID("round_id", req.RoundID),
					attr.String("time_string", *req.StartTime),
					attr.String("timezone", *req.Timezone),
				)

				// Use time parser exactly like create round
				parsedTimeUnix, err := timeParser.ParseUserTimeInput(
					*req.StartTime,
					roundtypes.Timezone(*req.Timezone),
					clock,
				)
				if err != nil {
					s.logger.ErrorContext(ctx, "Time parsing failed for round update",
						attr.ExtractCorrelationID(ctx),
						attr.RoundID("round_id", req.RoundID),
						attr.String("time_string", *req.StartTime),
						attr.Error(err),
					)
					s.metrics.RecordTimeParsingError(ctx)
					errs = append(errs, fmt.Sprintf("time parsing failed: %v", err))
				} else {
					// Convert and validate parsed time (like create round)
					parsedTime := time.Unix(parsedTimeUnix, 0).UTC()
					currentTime := clock.NowUTC()

					if parsedTime.Before(currentTime) {
						s.logger.InfoContext(ctx, "Parsed time is in the past",
							attr.ExtractCorrelationID(ctx),
							attr.RoundID("round_id", req.RoundID),
							attr.String("parsed_time", parsedTime.Format(time.RFC3339)),
						)
						s.metrics.RecordValidationError(ctx)
						errs = append(errs, "start time cannot be in the past")
					} else {
						// Success - store parsed time
						startTime := sharedtypes.StartTime(parsedTime)
						parsedStartTime = &startTime
						s.metrics.RecordTimeParsingSuccess(ctx)

						s.logger.InfoContext(ctx, "Time parsing successful for round update",
							attr.ExtractCorrelationID(ctx),
							attr.RoundID("round_id", req.RoundID),
							attr.String("parsed_time_utc", parsedTime.Format(time.RFC3339)),
						)
					}
				}
			}
		}

		// Check for validation errors (like create round)
		if len(errs) > 0 {
			s.logger.ErrorContext(ctx, "Round update validation failed",
				attr.ExtractCorrelationID(ctx),
				attr.RoundID("round_id", req.RoundID),
				attr.Any("validation_errors", errs),
			)
			s.metrics.RecordValidationError(ctx)

			return results.FailureResult[*roundtypes.UpdateRoundResult, error](fmt.Errorf("validation failed: %s", strings.Join(errs, "; "))), nil
		}

		s.metrics.RecordValidationSuccess(ctx)
		s.logger.InfoContext(ctx, "Round update validation successful",
			attr.ExtractCorrelationID(ctx),
			attr.RoundID("round_id", req.RoundID),
		)

		partialRound := &roundtypes.Round{
			ID: req.RoundID,
		}
		if parsedStartTime != nil {
			partialRound.StartTime = parsedStartTime
		}
		if req.Title != nil {
			partialRound.Title = roundtypes.Title(*req.Title)
		}

		return results.SuccessResult[*roundtypes.UpdateRoundResult, error](&roundtypes.UpdateRoundResult{
			Round: partialRound, // Carries parsed start time
		}), nil
	})
}

// Backwards-compatible wrapper using the real clock.
func (s *RoundService) ValidateRoundUpdate(ctx context.Context, req *roundtypes.UpdateRoundRequest, timeParser roundtime.TimeParserInterface) (UpdateRoundResult, error) {
	return s.ValidateRoundUpdateWithClock(ctx, req, timeParser, roundutil.RealClock{})
}

// UpdateRoundEntity updates the round entity with validated and parsed values
func (s *RoundService) UpdateRoundEntity(ctx context.Context, req *roundtypes.UpdateRoundRequest) (UpdateRoundResult, error) {
	return withTelemetry(s, ctx, "UpdateRoundEntity", req.RoundID, func(ctx context.Context) (UpdateRoundResult, error) {
		roundID := req.RoundID
		guildID := req.GuildID

		s.logger.InfoContext(ctx, "Updating round entity",
			attr.RoundID("round_id", roundID),
		)

		// Step 1: Fetch current round to preserve required fields
		currentRound, err := s.repo.GetRound(ctx, s.db, guildID, roundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to fetch current round before update",
				attr.RoundID("round_id", roundID),
				attr.Error(err),
			)
			s.metrics.RecordDBOperationError(ctx, "GetRound")
			// Return as failure result so it can be published as an error event
			return results.FailureResult[*roundtypes.UpdateRoundResult](fmt.Errorf("failed to fetch round: %w", err)), nil
		}

		// Step 2: Create update object starting with current required fields
		updateRound := &roundtypes.Round{
			ID:              roundID,
			Title:           currentRound.Title,
			Description:     currentRound.Description,
			Location:        currentRound.Location,
			StartTime:       currentRound.StartTime,
			EventType:       currentRound.EventType,
			ImportID:        currentRound.ImportID,
			ImportStatus:    currentRound.ImportStatus,
			ImportType:      currentRound.ImportType,
			FileData:        currentRound.FileData,
			FileName:        currentRound.FileName,
			UDiscURL:        currentRound.UDiscURL,
			ImportNotes:     currentRound.ImportNotes,
			ImportError:     currentRound.ImportError,
			ImportErrorCode: currentRound.ImportErrorCode,
			ImportedAt:      currentRound.ImportedAt,
			ImportUserID:    currentRound.ImportUserID,
			ImportChannelID: currentRound.ImportChannelID,
			Finalized:       currentRound.Finalized,
			CreatedBy:       currentRound.CreatedBy,
			State:           currentRound.State,
			EventMessageID:  currentRound.EventMessageID,
			Participants:    currentRound.Participants,
		}

		var updatedFields []string

		// Step 3: Overwrite only the fields provided by the user
		if req.Title != nil && *req.Title != string(currentRound.Title) {
			updateRound.Title = roundtypes.Title(*req.Title)
			updatedFields = append(updatedFields, "title")
		}
		if req.Description != nil {
			// Handle cases where currentRound.Description may be a value type (zero-value) instead of a pointer.
			if string(currentRound.Description) == "" || *req.Description != string(currentRound.Description) {
				updateRound.Description = roundtypes.Description(*req.Description)
				updatedFields = append(updatedFields, "description")
			}
		}
		if req.Location != nil {
			if string(currentRound.Location) == "" || *req.Location != string(currentRound.Location) {
				updateRound.Location = roundtypes.Location(*req.Location)
				updatedFields = append(updatedFields, "location")
			}
		}
		if req.ParsedStartTime != nil {
			if currentRound.StartTime == nil || *req.ParsedStartTime != *currentRound.StartTime {
				updateRound.StartTime = new(sharedtypes.StartTime)
				*updateRound.StartTime = *req.ParsedStartTime
				updatedFields = append(updatedFields, "start_time")
			}
		} else if req.StartTime != nil && req.ParsedStartTime == nil {
			// Should have been parsed already, but if not, logic error?
			// ValidateRoundUpdate should have populated ParsedStartTime if StartTime was provided.
			// If it wasn't provided, this block is skipped.
		}

		if req.EventType != nil {
			if currentRound.EventType == nil || *req.EventType != *currentRound.EventType {
				updateRound.EventType = req.EventType
				updatedFields = append(updatedFields, "event_type")
			}
		}

		// Step 4: Ensure there is at least one field to update
		if len(updatedFields) == 0 {
			s.logger.WarnContext(ctx, "No fields to update after processing",
				attr.RoundID("round_id", roundID),
			)
			return results.FailureResult[*roundtypes.UpdateRoundResult, error](fmt.Errorf("no valid fields to update")), nil
		}

		// Step 5: Update in DB
		updatedRound, err := s.repo.UpdateRound(ctx, s.db, guildID, roundID, updateRound)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to update round entity",
				attr.RoundID("round_id", roundID),
				attr.Error(err),
			)
			s.metrics.RecordDBOperationError(ctx, "UpdateRound")
			return results.OperationResult[*roundtypes.UpdateRoundResult, error]{}, err
		}

		s.metrics.RecordDBOperationSuccess(ctx, "UpdateRound")
		s.logger.InfoContext(ctx, "Round entity updated successfully",
			attr.RoundID("round_id", roundID),
			attr.Any("updated_fields", updatedFields),
		)

		// Step 6: Ensure GuildID is always set on returned round
		updatedRound.GuildID = guildID

		return results.SuccessResult[*roundtypes.UpdateRoundResult, error](&roundtypes.UpdateRoundResult{
			Round: updatedRound,
		}), nil
	})
}

// UpdateScheduledRoundEvents updates the scheduled events for a round.
func (s *RoundService) UpdateScheduledRoundEvents(ctx context.Context, req *roundtypes.UpdateScheduledRoundEventsRequest) (UpdateScheduledRoundEventsResult, error) {
	return withTelemetry(s, ctx, "UpdateScheduledRoundEvents", req.RoundID, func(ctx context.Context) (UpdateScheduledRoundEventsResult, error) {
		if req.GuildID == "" {
			s.logger.ErrorContext(ctx, "GuildID missing in RoundScheduleUpdatePayload; aborting reschedule to prevent orphaned jobs",
				attr.RoundID("round_id", req.RoundID),
			)
			return results.FailureResult[bool, error](fmt.Errorf("guild id missing for scheduled round update")), nil
		}

		// Step 1: Cancel existing scheduled events
		s.logger.InfoContext(ctx, "Cancelling existing scheduled jobs",
			attr.RoundID("round_id", req.RoundID),
		)

		if err := s.queueService.CancelRoundJobs(ctx, req.RoundID); err != nil {
			s.logger.ErrorContext(ctx, "CRITICAL: Failed to cancel existing scheduled jobs",
				attr.RoundID("round_id", req.RoundID),
				attr.Error(err),
			)
			return results.OperationResult[bool, error]{}, err
		}

		s.logger.InfoContext(ctx, "Successfully cancelled existing scheduled jobs",
			attr.RoundID("round_id", req.RoundID),
		)

		// Step 2: Get EventMessageID for rescheduling
		eventMessageID, err := s.repo.GetEventMessageID(ctx, s.db, req.GuildID, req.RoundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get EventMessageID for rescheduling",
				attr.RoundID("round_id", req.RoundID),
				attr.Error(err),
			)
			return results.OperationResult[bool, error]{}, err
		}

		// Step 3: Get current round data to preserve fields not being updated
		currentRound, err := s.repo.GetRound(ctx, s.db, req.GuildID, req.RoundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get current round data for rescheduling",
				attr.RoundID("round_id", req.RoundID),
				attr.Error(err),
			)
			return results.OperationResult[bool, error]{}, err
		}

		// Step 4: Determine final values (updated or preserved)
		finalTitle := string(currentRound.Title)
		if req.Title != nil {
			finalTitle = *req.Title
		}

		finalLocation := string(currentRound.Location)
		if req.Location != nil {
			finalLocation = *req.Location
		}

		// Step 5: Schedule new events
		now := time.Now().UTC()

		// Check if StartTime is provided
		if req.StartTime == nil {
			return results.FailureResult[bool, error](fmt.Errorf("start time is required for rescheduling events")), nil
		}

		startTimeUTC := req.StartTime.AsTime().UTC()

		s.logger.InfoContext(ctx, "Time comparison debug for rescheduling",
			attr.RoundID("round_id", req.RoundID),
			attr.Time("start_time_utc", startTimeUTC),
			attr.Time("current_time_utc", now),
		)

		// Only proceed if the round start time is in the future
		if !startTimeUTC.After(now) {
			s.logger.WarnContext(ctx, "Round start time is not in the future, cannot reschedule events",
				attr.RoundID("round_id", req.RoundID),
				attr.Time("start_time", startTimeUTC),
				attr.Time("current_time", now),
			)
			return results.FailureResult[bool, error](fmt.Errorf("Round start time must be in the future")), nil
		}

		// Calculate reminder time (1 hour before the round starts) in UTC
		reminderTimeUTC := startTimeUTC.Add(-1 * time.Hour)

		// Only schedule reminder if there's enough time (reminder time is in the future)
		if reminderTimeUTC.After(now) {
			s.logger.InfoContext(ctx, "Rescheduling 1-hour reminder",
				attr.RoundID("round_id", req.RoundID),
				attr.Time("reminder_time", reminderTimeUTC),
			)

			// Prepare UserIDs for reminder
			var userIDs []sharedtypes.DiscordID
			if currentRound.Participants != nil {
				for _, p := range currentRound.Participants {
					userIDs = append(userIDs, p.UserID)
				}
			}

			// Reminder payload
			reminderPayload := roundevents.DiscordReminderPayloadV1{
				GuildID:        req.GuildID,
				RoundID:        req.RoundID,
				ReminderType:   "1h",
				RoundTitle:     roundtypes.Title(finalTitle),
				Location:       roundtypes.Location(finalLocation),
				StartTime:      req.StartTime,
				UserIDs:        userIDs,
				EventMessageID: eventMessageID,
				DiscordGuildID: string(req.GuildID),
			}

			// Enrich with guild config to embed DiscordChannelID if available
			if cfg := s.getGuildConfigForEnrichment(ctx, req.GuildID); cfg != nil && cfg.EventChannelID != "" {
				reminderPayload.DiscordChannelID = cfg.EventChannelID
				s.logger.DebugContext(ctx, "Embedding event channel ID into rescheduled reminder data",
					attr.String("channel_id", cfg.EventChannelID),
				)
			}

			if err := s.queueService.ScheduleRoundReminder(ctx, req.GuildID, req.RoundID, reminderTimeUTC, reminderPayload); err != nil {
				s.logger.ErrorContext(ctx, "Failed to reschedule reminder",
					attr.RoundID("round_id", req.RoundID),
					attr.Error(err),
				)
				return results.OperationResult[bool, error]{}, err
			}

			s.logger.InfoContext(ctx, "Successfully rescheduled 1-hour reminder",
				attr.RoundID("round_id", req.RoundID),
				attr.Time("reminder_time", reminderTimeUTC),
			)
		} else {
			s.logger.InfoContext(ctx, "Skipping 1-hour reminder for rescheduling - not enough time",
				attr.RoundID("round_id", req.RoundID),
				attr.Time("start_time", startTimeUTC),
				attr.Time("current_time", now),
				attr.Time("reminder_time", reminderTimeUTC),
			)
		}

		// Schedule the round start event
		s.logger.InfoContext(ctx, "Rescheduling round start event",
			attr.RoundID("round_id", req.RoundID),
			attr.Time("start_time", startTimeUTC),
		)

		// startPayload := roundevents.RoundStartedPayloadV1{
		// 	GuildID:   req.GuildID,
		// 	RoundID:   req.RoundID,
		// 	Title:     roundtypes.Title(finalTitle),
		// 	Location:  roundtypes.Location(finalLocation),
		// 	StartTime: req.StartTime,
		// }

		// Enrich with config if available (checking again as it might have been skipped in reminder block)
		// if cfg := s.getGuildConfigForEnrichment(ctx, req.GuildID); cfg != nil && cfg.EventChannelID != "" {
		// 	startPayload.ChannelID = cfg.EventChannelID
		// }

		// if err := s.queueService.ScheduleRoundStart(ctx, req.GuildID, req.RoundID, startTimeUTC, startPayload); err != nil {
		// 	s.logger.ErrorContext(ctx, "Failed to reschedule round start",
		// 		attr.RoundID("round_id", req.RoundID),
		// 		attr.Error(err),
		// 	)
		// 	return results.OperationResult[bool, error]{}, err
		// }
		s.logger.InfoContext(ctx, "Skipping round start rescheduling (disabled per configuration/request)",
			attr.RoundID("round_id", req.RoundID),
			attr.Time("start_time", startTimeUTC),
		)

		s.logger.InfoContext(ctx, "Round events rescheduled successfully",
			attr.RoundID("round_id", req.RoundID),
			attr.Time("reminder_time", reminderTimeUTC),
			attr.Time("start_time", startTimeUTC),
		)

		return results.SuccessResult[bool, error](true), nil
	})
}
