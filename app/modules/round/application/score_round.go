package roundservice

import (
	"context"
	"fmt"
	"strings"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

// ValidateScoreUpdateRequest validates the score update request.
func (s *RoundService) ValidateScoreUpdateRequest(ctx context.Context, payload roundevents.ScoreUpdateRequestPayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "ValidateScoreUpdateRequest", func() (RoundOperationResult, error) {
		var errs []string
		if payload.RoundID == sharedtypes.RoundID(uuid.Nil) { // Check if RoundID is zero
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
			s.logger.Error("Score update request validation failed",
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

		s.logger.Info("Score update request validated",
			attr.RoundID("round_id", payload.RoundID),
		)

		return RoundOperationResult{
			Success: roundevents.ScoreUpdateValidatedPayload{
				ScoreUpdateRequestPayload: payload,
			},
		}, nil
	})
}

// UpdateParticipantScore updates the participant's score in the database.
func (s *RoundService) UpdateParticipantScore(ctx context.Context, payload roundevents.ScoreUpdateValidatedPayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "UpdateParticipantScore", func() (RoundOperationResult, error) {
		err := s.RoundDB.UpdateParticipantScore(ctx, payload.ScoreUpdateRequestPayload.RoundID, payload.ScoreUpdateRequestPayload.Participant, *payload.ScoreUpdateRequestPayload.Score)
		if err != nil {
			s.logger.Error("Failed to update participant score",
				attr.RoundID("round_id", payload.ScoreUpdateRequestPayload.RoundID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: roundevents.RoundScoreUpdateErrorPayload{
					ScoreUpdateRequest: &payload.ScoreUpdateRequestPayload,
					Error:              err.Error(),
				},
			}, err
		}

		// Get the round information, including the EventMessageID
		round, err := s.RoundDB.GetRound(ctx, payload.ScoreUpdateRequestPayload.RoundID)
		if err != nil {
			s.logger.Error("Failed to get round",
				attr.RoundID("round_id", payload.ScoreUpdateRequestPayload.RoundID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: payload.ScoreUpdateRequestPayload.RoundID,
					Error:   err.Error(),
				},
			}, err
		}

		s.logger.Info("Participant score updated in database",
			attr.RoundID("round_id", payload.ScoreUpdateRequestPayload.RoundID),
			attr.String("participant_id", string(payload.ScoreUpdateRequestPayload.Participant)),
			attr.Int("score", int(*payload.ScoreUpdateRequestPayload.Score)),
		)

		return RoundOperationResult{
			Success: roundevents.ParticipantScoreUpdatedPayload{
				RoundID:        payload.ScoreUpdateRequestPayload.RoundID,
				Participant:    payload.ScoreUpdateRequestPayload.Participant,
				Score:          *payload.ScoreUpdateRequestPayload.Score,
				EventMessageID: &round.EventMessageID,
			},
		}, nil
	})
}

// CheckAllScoresSubmitted checks if all participants in the round have submitted scores.
func (s *RoundService) CheckAllScoresSubmitted(ctx context.Context, payload roundevents.ParticipantScoreUpdatedPayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "CheckAllScoresSubmitted", func() (RoundOperationResult, error) {
		allScoresSubmitted, err := s.checkIfAllScoresSubmitted(ctx, payload.RoundID)
		if err != nil {
			s.logger.Error("Failed to check if all scores have been submitted",
				attr.RoundID("round_id", payload.RoundID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: payload.RoundID,
					Error:   err.Error(),
				},
			}, err
		}

		if allScoresSubmitted {
			s.logger.Info("All scores submitted for round",
				attr.RoundID("round_id", payload.RoundID),
				attr.RoundID("event_message_id", *payload.EventMessageID),
			)

			return RoundOperationResult{
				Success: roundevents.AllScoresSubmittedPayload{
					RoundID:        payload.RoundID,
					EventMessageID: payload.EventMessageID,
				},
			}, nil
		} else {
			s.logger.Info("Not all scores submitted yet, updating Discord",
				attr.RoundID("round_id", payload.RoundID),
				attr.String("participant_id", string(payload.Participant)),
				attr.Int("score", int(payload.Score)),
				attr.RoundID("event_message_id", *payload.EventMessageID),
			)

			return RoundOperationResult{
				Success: roundevents.NotAllScoresSubmittedPayload{
					RoundID:        payload.RoundID,
					Participant:    payload.Participant,
					Score:          payload.Score,
					EventMessageID: *payload.EventMessageID,
				},
			}, nil
		}
	})
}

// CheckIfAllScoresSubmitted checks if all participants in the round have submitted scores.
func (s *RoundService) checkIfAllScoresSubmitted(ctx context.Context, roundID sharedtypes.RoundID) (bool, error) {
	participants, err := s.RoundDB.GetParticipants(ctx, roundID)
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
