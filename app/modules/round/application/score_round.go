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

// ValidateScoreUpdateRequest validates the score update request.
// Multi-guild: require guildID for all round operations
func (s *RoundService) ValidateScoreUpdateRequest(ctx context.Context, payload roundevents.ScoreUpdateRequestPayloadV1) (RoundOperationResult, error) {
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
		// Note: GuildID may be absent on some incoming events; allow validation to pass
		// and rely on downstream handlers or DB operations to resolve or enforce it.
		// Add more validation rules as needed...

		if len(errs) > 0 {
			err := fmt.Errorf("validation errors: %s", strings.Join(errs, "; "))
			s.logger.ErrorContext(ctx, "Score update request validation failed",
				attr.RoundID("round_id", payload.RoundID),
				attr.String("guild_id", string(payload.GuildID)),
				attr.String("participant", string(payload.Participant)),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: &roundevents.RoundScoreUpdateErrorPayloadV1{
					GuildID:            payload.GuildID,
					ScoreUpdateRequest: &payload,
					Error:              err.Error(),
				},
			}, nil
		}

		s.logger.InfoContext(ctx, "Score update request validated",
			attr.RoundID("round_id", payload.RoundID),
		)

		return RoundOperationResult{
			Success: &roundevents.ScoreUpdateValidatedPayloadV1{
				GuildID:                   payload.GuildID,
				ScoreUpdateRequestPayload: payload,
			},
		}, nil
	})
}

// UpdateParticipantScore updates the participant's score in the database and publishes an event with the full participant list.
func (s *RoundService) UpdateParticipantScore(ctx context.Context, payload roundevents.ScoreUpdateValidatedPayloadV1) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "UpdateParticipantScore", payload.ScoreUpdateRequestPayload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		// Update the participant's score in the database
		err := s.RoundDB.UpdateParticipantScore(ctx, payload.ScoreUpdateRequestPayload.GuildID, payload.ScoreUpdateRequestPayload.RoundID, payload.ScoreUpdateRequestPayload.Participant, *payload.ScoreUpdateRequestPayload.Score)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to update participant score in DB",
				attr.RoundID("round_id", payload.ScoreUpdateRequestPayload.RoundID),
				attr.String("guild_id", string(payload.ScoreUpdateRequestPayload.GuildID)),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: &roundevents.RoundScoreUpdateErrorPayloadV1{
					GuildID:            payload.ScoreUpdateRequestPayload.GuildID,
					ScoreUpdateRequest: &payload.ScoreUpdateRequestPayload,
					Error:              "Failed to update score in database: " + err.Error(),
				},
			}, nil
		}

		// Fetch the full, updated list of participants for this round
		updatedParticipants, err := s.RoundDB.GetParticipants(ctx, payload.ScoreUpdateRequestPayload.GuildID, payload.ScoreUpdateRequestPayload.RoundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get updated participants after score update",
				attr.RoundID("round_id", payload.ScoreUpdateRequestPayload.RoundID),
				attr.String("guild_id", string(payload.ScoreUpdateRequestPayload.GuildID)),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
					GuildID: payload.ScoreUpdateRequestPayload.GuildID,
					RoundID: payload.ScoreUpdateRequestPayload.RoundID,
					Error:   "Failed to retrieve updated participants list after score update: " + err.Error(),
				},
			}, nil
		}

		// Fetch round details to get ChannelID and EventMessageID
		round, err := s.RoundDB.GetRound(ctx, payload.ScoreUpdateRequestPayload.GuildID, payload.ScoreUpdateRequestPayload.RoundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get round details after score update",
				attr.RoundID("round_id", payload.ScoreUpdateRequestPayload.RoundID),
				attr.String("guild_id", string(payload.ScoreUpdateRequestPayload.GuildID)),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
					GuildID: payload.ScoreUpdateRequestPayload.GuildID,
					RoundID: payload.ScoreUpdateRequestPayload.RoundID,
					Error:   "Failed to retrieve round details for event payload: " + err.Error(),
				},
			}, nil
		}

		s.logger.InfoContext(ctx, "Participant score updated in database and fetched updated participants",
			attr.RoundID("round_id", payload.ScoreUpdateRequestPayload.RoundID),
			attr.String("guild_id", string(payload.ScoreUpdateRequestPayload.GuildID)),
			attr.String("participant_id", string(payload.ScoreUpdateRequestPayload.Participant)),
			attr.Int("score", int(*payload.ScoreUpdateRequestPayload.Score)),
			attr.Int("updated_participant_count", len(updatedParticipants)),
		)

		// Publish the event with the full list of updated participants
		return RoundOperationResult{
			Success: &roundevents.ParticipantScoreUpdatedPayloadV1{
				GuildID:        payload.ScoreUpdateRequestPayload.GuildID,
				RoundID:        payload.ScoreUpdateRequestPayload.RoundID,
				Participant:    payload.ScoreUpdateRequestPayload.Participant,
				Score:          *payload.ScoreUpdateRequestPayload.Score,
				EventMessageID: round.EventMessageID,
				Participants:   updatedParticipants,
			},
		}, nil
	})
}

// CheckAllScoresSubmittedResult is a custom struct to return data from CheckAllScoresSubmitted.
type CheckAllScoresSubmittedResult struct {
	AllScoresSubmitted bool
	PayloadData        interface{} // Will hold AllScoresSubmittedPayload or NotAllScoresSubmittedPayload data
}

// CheckAllScoresSubmitted checks if all participants in the round have submitted scores.
func (s *RoundService) CheckAllScoresSubmitted(
	ctx context.Context,
	payload roundevents.ParticipantScoreUpdatedPayloadV1,
) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "CheckAllScoresSubmitted", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		for _, p := range payload.Participants {
			if p.Score == nil {
				return RoundOperationResult{
					Success: &roundevents.NotAllScoresSubmittedPayload{
						GuildID:        payload.GuildID,
						RoundID:        payload.RoundID,
						Participant:    payload.Participant,
						Score:          payload.Score,
						EventMessageID: payload.EventMessageID,
						Participants:   payload.Participants,
					},
				}, nil
			}
		}

		// Only happens ONCE
		round, err := s.RoundDB.GetRound(ctx, payload.GuildID, payload.RoundID)
		if err != nil {
			return RoundOperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
					GuildID: payload.GuildID,
					RoundID: payload.RoundID,
					Error:   err.Error(),
				},
			}, nil
		}

		round.Participants = payload.Participants

		return RoundOperationResult{
			Success: &roundevents.AllScoresSubmittedPayloadV1{
				GuildID:        payload.GuildID,
				RoundID:        payload.RoundID,
				EventMessageID: payload.EventMessageID,
				RoundData:      *round,
				Participants:   payload.Participants,
			},
		}, nil
	})
}

// CheckIfAllScoresSubmitted checks if all participants in the round have submitted scores.
// func (s *RoundService) checkIfAllScoresSubmitted(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (bool, error) {
// 	participants, err := s.RoundDB.GetParticipants(ctx, guildID, roundID)
// 	if err != nil {
// 		return false, err
// 	}

// 	for _, p := range participants {
// 		if p.Score == nil {
// 			return false, nil
// 		}
// 	}

// 	return true, nil
// }
