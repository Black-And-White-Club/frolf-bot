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
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"github.com/google/uuid"
)

// ValidateAndProcessRoundUpdate validates and processes round update with time parsing (like create round)
func (s *RoundService) ValidateAndProcessRoundUpdateWithClock(ctx context.Context, payload roundevents.UpdateRoundRequestedPayload, timeParser roundtime.TimeParserInterface, clock roundutil.Clock) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "ValidateAndProcessRoundUpdate", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Validating and processing round update request",
			attr.ExtractCorrelationID(ctx),
			attr.RoundID("round_id", payload.RoundID),
		)

		var errs []string

		// Basic validation checks (like create round)
		if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
			errs = append(errs, "round ID cannot be zero")
		}

		if payload.Title == nil && payload.Description == nil && payload.Location == nil && payload.StartTime == nil {
			errs = append(errs, "at least one field to update must be provided")
		}

		// Process time string if provided (exactly like create round) with nil-safe timezone handling
		var parsedStartTime *sharedtypes.StartTime
		if payload.StartTime != nil && *payload.StartTime != "" {
			if payload.Timezone == nil || *payload.Timezone == "" {
				errs = append(errs, "timezone is required when providing start time")
			} else {
				s.logger.InfoContext(ctx, "Processing time string for round update",
					attr.ExtractCorrelationID(ctx),
					attr.RoundID("round_id", payload.RoundID),
					attr.String("time_string", *payload.StartTime),
					attr.String("timezone", string(*payload.Timezone)),
				)

				// Use time parser exactly like create round
				parsedTimeUnix, err := timeParser.ParseUserTimeInput(
					*payload.StartTime,
					*payload.Timezone,
					clock,
				)
				if err != nil {
					s.logger.ErrorContext(ctx, "Time parsing failed for round update",
						attr.ExtractCorrelationID(ctx),
						attr.RoundID("round_id", payload.RoundID),
						attr.String("time_string", *payload.StartTime),
						attr.Error(err),
					)
					s.metrics.RecordTimeParsingError(ctx)
					errs = append(errs, fmt.Sprintf("time parsing failed: %v", err))
				} else {
					// Convert and validate parsed time (like create round)
					parsedTime := time.Unix(parsedTimeUnix, 0).UTC()
					currentTime := time.Now().UTC()

					if parsedTime.Before(currentTime) {
						s.logger.InfoContext(ctx, "Parsed time is in the past",
							attr.ExtractCorrelationID(ctx),
							attr.RoundID("round_id", payload.RoundID),
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
							attr.RoundID("round_id", payload.RoundID),
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
				attr.RoundID("round_id", payload.RoundID),
				attr.Any("validation_errors", errs),
			)
			s.metrics.RecordValidationError(ctx)

			return RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: nil,
					Error:              fmt.Sprintf("validation failed: %s", strings.Join(errs, "; ")),
				},
			}, nil // Return nil error like create round
		}

		s.metrics.RecordValidationSuccess(ctx)
		s.logger.InfoContext(ctx, "Round update validation successful",
			attr.ExtractCorrelationID(ctx),
			attr.RoundID("round_id", payload.RoundID),
		)

		// Build the validated payload for the next step
		validatedPayload := roundevents.RoundUpdateRequestPayload{
			GuildID:   payload.GuildID, // Copy GuildID for multi-tenant correctness
			RoundID:   payload.RoundID,
			UserID:    payload.UserID,
			StartTime: parsedStartTime, // Use parsed time (or nil if not provided)
			EventType: nil,             // Will be set later if needed
		}

		// Copy other fields if provided
		if payload.Title != nil {
			validatedPayload.Title = *payload.Title
		}
		if payload.Description != nil {
			validatedPayload.Description = payload.Description
		}
		if payload.Location != nil {
			validatedPayload.Location = payload.Location
		}

		// Return validation success payload (like create round pattern)
		return RoundOperationResult{
			Success: &roundevents.RoundUpdateValidatedPayload{
				RoundUpdateRequestPayload: validatedPayload,
			},
		}, nil
	})
}

// Backwards-compatible wrapper using the real clock.
func (s *RoundService) ValidateAndProcessRoundUpdate(ctx context.Context, payload roundevents.UpdateRoundRequestedPayload, timeParser roundtime.TimeParserInterface) (RoundOperationResult, error) {
	return s.ValidateAndProcessRoundUpdateWithClock(ctx, payload, timeParser, roundutil.RealClock{})
}

// UpdateRoundEntity updates the round entity with the validated and parsed values
func (s *RoundService) UpdateRoundEntity(ctx context.Context, payload roundevents.RoundUpdateValidatedPayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "UpdateRoundEntity", payload.RoundUpdateRequestPayload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Updating round entity",
			attr.RoundID("round_id", payload.RoundUpdateRequestPayload.RoundID),
		)

		// Create a round object with only the fields to update
		updateRound := &roundtypes.Round{
			ID: payload.RoundUpdateRequestPayload.RoundID,
		}

		var updatedFields []string

		// Only set fields that are being updated
		if payload.RoundUpdateRequestPayload.Title != "" {
			updateRound.Title = payload.RoundUpdateRequestPayload.Title
			updatedFields = append(updatedFields, "title")
		}
		if payload.RoundUpdateRequestPayload.Description != nil {
			updateRound.Description = payload.RoundUpdateRequestPayload.Description
			updatedFields = append(updatedFields, "description")
		}
		if payload.RoundUpdateRequestPayload.Location != nil {
			updateRound.Location = payload.RoundUpdateRequestPayload.Location
			updatedFields = append(updatedFields, "location")
		}
		if payload.RoundUpdateRequestPayload.StartTime != nil {
			updateRound.StartTime = payload.RoundUpdateRequestPayload.StartTime
			updatedFields = append(updatedFields, "start_time")
		}
		if payload.RoundUpdateRequestPayload.EventType != nil {
			updateRound.EventType = payload.RoundUpdateRequestPayload.EventType
			updatedFields = append(updatedFields, "event_type")
		}

		// Ensure we have something to update
		if len(updatedFields) == 0 {
			s.logger.WarnContext(ctx, "No fields to update after processing",
				attr.RoundID("round_id", payload.RoundUpdateRequestPayload.RoundID),
			)
			return RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: &payload.RoundUpdateRequestPayload,
					Error:              "no valid fields to update",
				},
			}, nil
		}

		s.logger.InfoContext(ctx, "Applying selective round updates",
			attr.ExtractCorrelationID(ctx),
			attr.RoundID("round_id", payload.RoundUpdateRequestPayload.RoundID),
			attr.Any("updated_fields", updatedFields),
		)

		// Update only the specified fields in the database - using the Round struct
		updatedRound, err := s.RoundDB.UpdateRound(ctx, payload.RoundUpdateRequestPayload.GuildID, payload.RoundUpdateRequestPayload.RoundID, updateRound)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to update round entity",
				attr.RoundID("round_id", payload.RoundUpdateRequestPayload.RoundID),
				attr.Error(err),
			)
			s.metrics.RecordDBOperationError(ctx, "UpdateRound")
			return RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: &payload.RoundUpdateRequestPayload,
					Error:              fmt.Sprintf("failed to update round in database: %v", err),
				},
			}, nil
		}

		s.metrics.RecordDBOperationSuccess(ctx, "UpdateRound")
		s.logger.InfoContext(ctx, "Round entity updated successfully",
			attr.RoundID("round_id", payload.RoundUpdateRequestPayload.RoundID),
			attr.Any("updated_fields", updatedFields),
		)

		return RoundOperationResult{
			Success: &roundevents.RoundEntityUpdatedPayload{
				Round: *updatedRound,
			},
		}, nil
	})
}

// UpdateScheduledRoundEvents updates the scheduled events for a round.
func (s *RoundService) UpdateScheduledRoundEvents(ctx context.Context, payload roundevents.RoundScheduleUpdatePayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "UpdateScheduledRoundEvents", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		if payload.GuildID == "" {
			s.logger.ErrorContext(ctx, "GuildID missing in RoundScheduleUpdatePayload; aborting reschedule to prevent orphaned jobs",
				attr.RoundID("round_id", payload.RoundID),
			)
			return RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: nil,
					Error:              "guild id missing for scheduled round update",
				},
			}, nil
		}
		s.logger.InfoContext(ctx, "Processing scheduled round update",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("title", string(payload.Title)),
			attr.Time("start_time", payload.StartTime.AsTime()),
		)

		// Step 1: Cancel existing scheduled events
		s.logger.InfoContext(ctx, "Cancelling existing scheduled jobs",
			attr.RoundID("round_id", payload.RoundID),
		)

		if err := s.QueueService.CancelRoundJobs(ctx, payload.RoundID); err != nil {
			s.logger.ErrorContext(ctx, "CRITICAL: Failed to cancel existing scheduled jobs",
				attr.RoundID("round_id", payload.RoundID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: nil,
					Error:              fmt.Sprintf("failed to cancel existing scheduled jobs: %v", err),
				},
			}, nil
		}

		s.logger.InfoContext(ctx, "Successfully cancelled existing scheduled jobs",
			attr.RoundID("round_id", payload.RoundID),
		)

		// Step 2: Get EventMessageID for rescheduling
		eventMessageID, err := s.RoundDB.GetEventMessageID(ctx, payload.GuildID, payload.RoundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get EventMessageID for rescheduling",
				attr.RoundID("round_id", payload.RoundID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: nil,
					Error:              fmt.Sprintf("failed to get EventMessageID: %v", err),
				},
			}, nil
		}

		// Step 3: Get current round data to preserve fields not being updated
		currentRound, err := s.RoundDB.GetRound(ctx, payload.GuildID, payload.RoundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get current round data for rescheduling",
				attr.RoundID("round_id", payload.RoundID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: nil,
					Error:              fmt.Sprintf("failed to get current round data: %v", err),
				},
			}, nil
		}

		// Step 4: Determine final values (updated or preserved)
		finalTitle := payload.Title
		if finalTitle == "" {
			finalTitle = currentRound.Title
		}

		finalLocation := payload.Location
		if finalLocation == nil {
			finalLocation = currentRound.Location
		}

		// Step 5: Schedule new events
		now := time.Now().UTC()

		// Check if StartTime is provided
		if payload.StartTime == nil {
			return RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: nil,
					Error:              "start time is required for rescheduling events",
				},
			}, nil
		}

		startTimeUTC := payload.StartTime.AsTime().UTC()

		s.logger.InfoContext(ctx, "Time comparison debug for rescheduling",
			attr.RoundID("round_id", payload.RoundID),
			attr.Time("start_time_utc", startTimeUTC),
			attr.Time("current_time_utc", now),
		)

		// Only proceed if the round start time is in the future
		if !startTimeUTC.After(now) {
			s.logger.WarnContext(ctx, "Round start time is not in the future, cannot reschedule events",
				attr.RoundID("round_id", payload.RoundID),
				attr.Time("start_time", startTimeUTC),
				attr.Time("current_time", now),
			)
			return RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: nil,
					Error:              "Round start time must be in the future",
				},
			}, nil
		}

		// Calculate reminder time (1 hour before the round starts) in UTC
		reminderTimeUTC := startTimeUTC.Add(-1 * time.Hour)

		// Only schedule reminder if there's enough time (reminder time is in the future)
		if reminderTimeUTC.After(now) {
			s.logger.InfoContext(ctx, "Rescheduling 1-hour reminder",
				attr.RoundID("round_id", payload.RoundID),
				attr.Time("reminder_time", reminderTimeUTC),
			)

			reminderPayload := roundevents.DiscordReminderPayload{
				GuildID:        payload.GuildID,
				RoundID:        payload.RoundID,
				ReminderType:   "1h",
				RoundTitle:     finalTitle,
				Location:       finalLocation,
				StartTime:      payload.StartTime,
				EventMessageID: eventMessageID,
			}
			// Enrich with guild config to embed DiscordChannelID if available
			if cfg := s.getGuildConfigForEnrichment(ctx, payload.GuildID); cfg != nil && cfg.EventChannelID != "" {
				reminderPayload.DiscordChannelID = cfg.EventChannelID
				s.logger.DebugContext(ctx, "Embedding event channel ID into rescheduled reminder payload",
					attr.String("channel_id", cfg.EventChannelID),
				)
			}

			if err := s.QueueService.ScheduleRoundReminder(ctx, payload.GuildID, payload.RoundID, reminderTimeUTC, reminderPayload); err != nil {
				s.logger.ErrorContext(ctx, "Failed to reschedule reminder",
					attr.RoundID("round_id", payload.RoundID),
					attr.Error(err),
				)
				return RoundOperationResult{
					Failure: &roundevents.RoundUpdateErrorPayload{
						RoundUpdateRequest: nil,
						Error:              err.Error(),
					},
				}, nil
			}

			s.logger.InfoContext(ctx, "Successfully rescheduled 1-hour reminder",
				attr.RoundID("round_id", payload.RoundID),
				attr.Time("reminder_time", reminderTimeUTC),
			)
		} else {
			s.logger.InfoContext(ctx, "Skipping 1-hour reminder for rescheduling - not enough time",
				attr.RoundID("round_id", payload.RoundID),
				attr.Time("start_time", startTimeUTC),
				attr.Time("current_time", now),
				attr.Time("reminder_time", reminderTimeUTC),
			)
		}

		// Schedule the round start event
		s.logger.InfoContext(ctx, "Rescheduling round start event",
			attr.RoundID("round_id", payload.RoundID),
			attr.Time("start_time", startTimeUTC),
		)

		startPayload := roundevents.RoundStartedPayload{
			GuildID:   payload.GuildID,
			RoundID:   payload.RoundID,
			Title:     finalTitle,
			Location:  finalLocation,
			StartTime: payload.StartTime,
		}
		if cfg := s.getGuildConfigForEnrichment(ctx, payload.GuildID); cfg != nil && cfg.EventChannelID != "" {
			// Only set if struct supports this field (guarded by build attempt)
			// (If RoundStartedPayload lacks DiscordChannelID in this version, this assignment will be a no-op compile removal.)
			// startPayload.DiscordChannelID = cfg.EventChannelID
		}

		if err := s.QueueService.ScheduleRoundStart(ctx, payload.GuildID, payload.RoundID, startTimeUTC, startPayload); err != nil {
			s.logger.ErrorContext(ctx, "Failed to reschedule round start",
				attr.RoundID("round_id", payload.RoundID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: nil,
					Error:              err.Error(),
				},
			}, nil
		}

		s.logger.InfoContext(ctx, "Round events rescheduled successfully",
			attr.RoundID("round_id", payload.RoundID),
			attr.Time("reminder_time", reminderTimeUTC),
			attr.Time("start_time", startTimeUTC),
		)

		// Return success with the update payload
		return RoundOperationResult{
			Success: &roundevents.RoundScheduleUpdatePayload{
				GuildID:   payload.GuildID,
				RoundID:   payload.RoundID,
				Title:     finalTitle,
				Location:  finalLocation,
				StartTime: payload.StartTime,
			},
		}, nil
	})
}
