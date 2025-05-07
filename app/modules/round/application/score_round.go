package roundservice

import (
	"context"
	"fmt"
	"strings"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr" // Ensure roundtypes is imported for RoundParticipant
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

// RoundOperationResult represents the result of a round service operation.
// Assuming this struct is defined elsewhere and includes Success interface{} and Failure interface{}

// ValidateScoreUpdateRequest validates the score update request.
func (s *RoundService) ValidateScoreUpdateRequest(ctx context.Context, payload roundevents.ScoreUpdateRequestPayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "ValidateScoreUpdateRequest", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		var errs []string
		if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
			errs = append(errs, "round ID cannot be zero")
		}
		if payload.Participant == "" {
			errs = append(errs, "participant Discord ID cannot be empty")
		}
		if payload.Score == nil {
			errs = append(errs, "score cannot be empty")
		}
		// Add more validation rules as needed...

		if len(errs) > 0 {
			err := fmt.Errorf("validation errors: %s", strings.Join(errs, "; "))
			s.logger.ErrorContext(ctx, "Score update request validation failed",
				attr.RoundID("round_id", payload.RoundID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: roundevents.RoundScoreUpdateErrorPayload{
					ScoreUpdateRequest: &payload,
					Error:              err.Error(),
				},
			}, err
		}

		s.logger.InfoContext(ctx, "Score update request validated",
			attr.RoundID("round_id", payload.RoundID),
		)

		return RoundOperationResult{
			Success: &roundevents.ScoreUpdateValidatedPayload{ // Return POINTER here
				ScoreUpdateRequestPayload: payload,
			},
		}, nil
	})
}

// UpdateParticipantScore updates the participant's score in the database and publishes an event with the full participant list.
func (s *RoundService) UpdateParticipantScore(ctx context.Context, payload roundevents.ScoreUpdateValidatedPayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "UpdateParticipantScore", payload.ScoreUpdateRequestPayload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		// Update the participant's score in the database
		err := s.RoundDB.UpdateParticipantScore(ctx, payload.ScoreUpdateRequestPayload.RoundID, payload.ScoreUpdateRequestPayload.Participant, *payload.ScoreUpdateRequestPayload.Score)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to update participant score in DB", // Added context to log
				attr.RoundID("round_id", payload.ScoreUpdateRequestPayload.RoundID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: roundevents.RoundScoreUpdateErrorPayload{
					ScoreUpdateRequest: &payload.ScoreUpdateRequestPayload,
					Error:              "Failed to update score in database: " + err.Error(),
				},
			}, fmt.Errorf("failed to update participant score in DB: %w", err) // Return error to handler
		}

		// Fetch the full, updated list of participants for this round
		updatedParticipants, err := s.RoundDB.GetParticipants(ctx, payload.ScoreUpdateRequestPayload.RoundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get updated participants after score update",
				attr.RoundID("round_id", payload.ScoreUpdateRequestPayload.RoundID),
				attr.Error(err),
			)
			// This is a critical failure for updating the Discord embed. Decide how to handle.
			// Returning a failure is appropriate as the Discord embed won't have the latest info.
			return RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{ // Using a general error payload or a specific one
					RoundID: payload.ScoreUpdateRequestPayload.RoundID,
					Error:   "Failed to retrieve updated participants list after score update: " + err.Error(),
				},
			}, fmt.Errorf("failed to get updated participants list: %w", err)
		}

		// Fetch round details to get ChannelID and EventMessageID
		round, err := s.RoundDB.GetRound(ctx, payload.ScoreUpdateRequestPayload.RoundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get round details after score update",
				attr.RoundID("round_id", payload.ScoreUpdateRequestPayload.RoundID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: payload.ScoreUpdateRequestPayload.RoundID,
					Error:   "Failed to retrieve round details for event payload: " + err.Error(),
				},
			}, fmt.Errorf("failed to get round details for event payload: %w", err)
		}

		s.logger.InfoContext(ctx, "Participant score updated in database and fetched updated participants",
			attr.RoundID("round_id", payload.ScoreUpdateRequestPayload.RoundID),
			attr.String("participant_id", string(payload.ScoreUpdateRequestPayload.Participant)),
			attr.Int("score", int(*payload.ScoreUpdateRequestPayload.Score)),
			attr.Int("updated_participant_count", len(updatedParticipants)), // Log count
		)

		// Publish the event with the full list of updated participants
		return RoundOperationResult{
			Success: &roundevents.ParticipantScoreUpdatedPayload{ // Return POINTER
				RoundID:        payload.ScoreUpdateRequestPayload.RoundID,
				Participant:    payload.ScoreUpdateRequestPayload.Participant,
				Score:          *payload.ScoreUpdateRequestPayload.Score,
				EventMessageID: round.EventMessageID,
				Participants:   updatedParticipants,
			},
		}, nil
	})
}

// CheckAllScoresSubmitted checks if all participants in the round have submitted scores.
// CheckAllScoresSubmittedResult is a custom struct to return data from CheckAllScoresSubmitted.
type CheckAllScoresSubmittedResult struct {
	AllScoresSubmitted bool
	PayloadData        interface{} // Will hold AllScoresSubmittedPayload or NotAllScoresSubmittedPayload data
}

// CheckAllScoresSubmitted checks if all participants in the round have submitted scores.
// It now *returns* data instead of publishing events.
func (s *RoundService) CheckAllScoresSubmitted(ctx context.Context, payload roundevents.ParticipantScoreUpdatedPayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "CheckAllScoresSubmitted", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) { // Use payload.RoundID
		allScoresSubmitted, err := s.checkIfAllScoresSubmitted(ctx, payload.RoundID) // Pass payload.RoundID
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to check if all scores have been submitted",
				attr.RoundID("round_id", payload.RoundID), // Use payload.RoundID
				attr.Error(err),
			)
			// Return a failure result if the check itself fails
			return RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: payload.RoundID, // Use payload.RoundID
					Error:   err.Error(),
				},
			}, fmt.Errorf("failed to check if all scores have been submitted: %w", err)
		}

		// Fetch the full, updated list of participants for this round
		// This is needed for both success payloads (AllScoresSubmittedPayload/NotAllScoresSubmittedPayload)
		updatedParticipants, err := s.RoundDB.GetParticipants(ctx, payload.RoundID) // Pass payload.RoundID
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get updated participants list for score check",
				attr.RoundID("round_id", payload.RoundID), // Use payload.RoundID
				attr.Error(err),
			)
			// Return a failure result if fetching participants fails
			return RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: payload.RoundID, // Use payload.RoundID
					Error:   "Failed to retrieve updated participants list for score check: " + err.Error(),
				},
			}, fmt.Errorf("failed to get updated participants list for score check: %w", err)
		}

		if allScoresSubmitted {
			s.logger.InfoContext(ctx, "All scores submitted for round",
				attr.RoundID("round_id", payload.RoundID), // Use payload.RoundID
			)
			// Return the payload data for AllScoresSubmitted as a struct
			return RoundOperationResult{
				Success: roundevents.AllScoresSubmittedPayload{ // Return VALUE here
					RoundID:        payload.RoundID,        // Use payload.RoundID
					EventMessageID: payload.EventMessageID, // Get from incoming payload
					Participants:   updatedParticipants,    // Include participants
					// Include other necessary round details for FinalizeRound service payload if needed
					// RoundData: ...,
				},
			}, nil
		} else {
			s.logger.InfoContext(ctx, "Not all scores submitted yet",
				attr.RoundID("round_id", payload.RoundID),                  // Use payload.RoundID
				attr.String("participant_id", string(payload.Participant)), // Use payload.Participant
				attr.Int("score", int(payload.Score)),                      // Use payload.Score
			)
			// Return the payload data for NotAllScoresSubmitted as a struct
			return RoundOperationResult{
				Success: roundevents.NotAllScoresSubmittedPayload{ // Return VALUE here
					RoundID:        payload.RoundID,        // Use payload.RoundID
					Participant:    payload.Participant,    // Get from incoming payload
					Score:          payload.Score,          // Get from incoming payload
					EventMessageID: payload.EventMessageID, // Get from incoming payload
					Participants:   updatedParticipants,    // Include participants
				},
			}, nil
		}
	})
}

// CheckIfAllScoresSubmitted checks if all participants in the round have submitted scores.
func (s *RoundService) checkIfAllScoresSubmitted(ctx context.Context, roundID sharedtypes.RoundID) (bool, error) {
	participants, err := s.RoundDB.GetParticipants(ctx, roundID) // Assumes GetParticipants returns []RoundParticipant with Score and TagNumber
	if err != nil {
		return false, err
	}

	for _, p := range participants {
		if p.Score == nil {
			return false, nil
		}
	}

	return true, nil
}
