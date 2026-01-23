package roundservice

import (
	"context"
	"fmt"
	"strings"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr" // Ensure roundtypes is imported for RoundParticipant
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
)

// ValidateScoreUpdateRequest validates the score update request.
// Multi-guild: require guildID for all round operations
func (s *RoundService) ValidateScoreUpdateRequest(ctx context.Context, payload roundevents.ScoreUpdateRequestPayloadV1) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "ValidateScoreUpdateRequest", payload.RoundID, func(ctx context.Context) (results.OperationResult, error) {
		var errs []string
		if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
			errs = append(errs, "round ID cannot be zero")
		}
		if payload.UserID == "" {
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
				attr.String("participant", string(payload.UserID)),
				attr.Error(err),
			)
			return results.OperationResult{
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

		return results.OperationResult{
			Success: &roundevents.ScoreUpdateValidatedPayloadV1{
				GuildID:                   payload.GuildID,
				ScoreUpdateRequestPayload: payload,
			},
		}, nil
	})
}

// UpdateParticipantScore updates the participant's score in the database and publishes an event with the full participant list.
func (s *RoundService) UpdateParticipantScore(ctx context.Context, payload roundevents.ScoreUpdateValidatedPayloadV1) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "UpdateParticipantScore", payload.ScoreUpdateRequestPayload.RoundID, func(ctx context.Context) (results.OperationResult, error) {
		// Update the participant's score in the database
		err := s.repo.UpdateParticipantScore(ctx, payload.ScoreUpdateRequestPayload.GuildID, payload.ScoreUpdateRequestPayload.RoundID, payload.ScoreUpdateRequestPayload.UserID, *payload.ScoreUpdateRequestPayload.Score)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to update participant score in DB",
				attr.RoundID("round_id", payload.ScoreUpdateRequestPayload.RoundID),
				attr.String("guild_id", string(payload.ScoreUpdateRequestPayload.GuildID)),
				attr.Error(err),
			)
			return results.OperationResult{
				Failure: &roundevents.RoundScoreUpdateErrorPayloadV1{
					GuildID:            payload.ScoreUpdateRequestPayload.GuildID,
					ScoreUpdateRequest: &payload.ScoreUpdateRequestPayload,
					Error:              "Failed to update score in database: " + err.Error(),
				},
			}, nil
		}

		// Fetch the full, updated list of participants for this round
		updatedParticipants, err := s.repo.GetParticipants(ctx, payload.ScoreUpdateRequestPayload.GuildID, payload.ScoreUpdateRequestPayload.RoundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get updated participants after score update",
				attr.RoundID("round_id", payload.ScoreUpdateRequestPayload.RoundID),
				attr.String("guild_id", string(payload.ScoreUpdateRequestPayload.GuildID)),
				attr.Error(err),
			)
			return results.OperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
					GuildID: payload.ScoreUpdateRequestPayload.GuildID,
					RoundID: payload.ScoreUpdateRequestPayload.RoundID,
					Error:   "Failed to retrieve updated participants list after score update: " + err.Error(),
				},
			}, nil
		}

		// Fetch round details to get ChannelID and EventMessageID
		round, err := s.repo.GetRound(ctx, payload.ScoreUpdateRequestPayload.GuildID, payload.ScoreUpdateRequestPayload.RoundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get round details after score update",
				attr.RoundID("round_id", payload.ScoreUpdateRequestPayload.RoundID),
				attr.String("guild_id", string(payload.ScoreUpdateRequestPayload.GuildID)),
				attr.Error(err),
			)
			return results.OperationResult{
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
			attr.String("participant_id", string(payload.ScoreUpdateRequestPayload.UserID)),
			attr.Int("score", int(*payload.ScoreUpdateRequestPayload.Score)),
			attr.Int("updated_participant_count", len(updatedParticipants)),
		)

		// Publish the event with the full list of updated participants
		return results.OperationResult{
			Success: &roundevents.ParticipantScoreUpdatedPayloadV1{
				GuildID:        payload.ScoreUpdateRequestPayload.GuildID,
				RoundID:        payload.ScoreUpdateRequestPayload.RoundID,
				UserID:         payload.ScoreUpdateRequestPayload.UserID,
				Score:          *payload.ScoreUpdateRequestPayload.Score,
				ChannelID:      payload.ScoreUpdateRequestPayload.ChannelID,
				EventMessageID: round.EventMessageID,
				Participants:   updatedParticipants,
			},
		}, nil
	})
}

// UpdateParticipantScoresBulk updates multiple participant scores in a single operation.
func (s *RoundService) UpdateParticipantScoresBulk(ctx context.Context, payload roundevents.ScoreBulkUpdateRequestPayloadV1) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "UpdateParticipantScoresBulk", payload.RoundID, func(ctx context.Context) (results.OperationResult, error) {
		if len(payload.Updates) == 0 {
			return results.OperationResult{
				Failure: &roundevents.RoundScoreUpdateErrorPayloadV1{
					GuildID:            payload.GuildID,
					ScoreUpdateRequest: nil,
					Error:              "bulk score update contains no updates",
				},
			}, nil
		}

		updatesByUser := make(map[sharedtypes.DiscordID]sharedtypes.Score, len(payload.Updates))
		for _, update := range payload.Updates {
			if update.Score == nil {
				return results.OperationResult{
					Failure: &roundevents.RoundScoreUpdateErrorPayloadV1{
						GuildID:            payload.GuildID,
						ScoreUpdateRequest: &update,
						Error:              "bulk score update contains empty score",
					},
				}, nil
			}
			updatesByUser[update.UserID] = *update.Score
		}

		participants, err := s.repo.GetParticipants(ctx, payload.GuildID, payload.RoundID)
		if err != nil {
			return results.OperationResult{
				Failure: &roundevents.RoundScoreUpdateErrorPayloadV1{
					GuildID:            payload.GuildID,
					ScoreUpdateRequest: &payload.Updates[0],
					Error:              "failed to fetch participants: " + err.Error(),
				},
			}, nil
		}

		updatedParticipants := make([]roundtypes.Participant, 0, len(participants))
		applied := 0
		for _, participant := range participants {
			if score, ok := updatesByUser[participant.UserID]; ok {
				s := score
				participant.Score = &s
				applied++
			}
			updatedParticipants = append(updatedParticipants, participant)
		}

		if applied != len(updatesByUser) {
			return results.OperationResult{
				Failure: &roundevents.RoundScoreUpdateErrorPayloadV1{
					GuildID:            payload.GuildID,
					ScoreUpdateRequest: &payload.Updates[0],
					Error:              "bulk score update includes users not in round",
				},
			}, nil
		}

		updates := []roundtypes.RoundUpdate{{
			RoundID:      payload.RoundID,
			Participants: updatedParticipants,
		}}
		if err := s.repo.UpdateRoundsAndParticipants(ctx, payload.GuildID, updates); err != nil {
			return results.OperationResult{
				Failure: &roundevents.RoundScoreUpdateErrorPayloadV1{
					GuildID:            payload.GuildID,
					ScoreUpdateRequest: &payload.Updates[0],
					Error:              "failed to update participant scores: " + err.Error(),
				},
			}, nil
		}

		round, err := s.repo.GetRound(ctx, payload.GuildID, payload.RoundID)
		if err != nil {
			return results.OperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
					GuildID: payload.GuildID,
					RoundID: payload.RoundID,
					Error:   "failed to fetch round details for bulk update: " + err.Error(),
				},
			}, nil
		}

		firstUpdate := payload.Updates[0]
		if firstUpdate.Score == nil {
			return results.OperationResult{
				Failure: &roundevents.RoundScoreUpdateErrorPayloadV1{
					GuildID:            payload.GuildID,
					ScoreUpdateRequest: &firstUpdate,
					Error:              "bulk score update missing score",
				},
			}, nil
		}

		channelID := payload.ChannelID
		messageID := payload.MessageID
		if channelID == "" {
			channelID = firstUpdate.ChannelID
		}
		if messageID == "" {
			messageID = firstUpdate.MessageID
		}

		eventMessageID := round.EventMessageID
		if messageID != "" {
			eventMessageID = messageID
		}

		return results.OperationResult{
			Success: &roundevents.RoundScoresBulkUpdatedPayloadV1{
				GuildID:        payload.GuildID,
				RoundID:        payload.RoundID,
				ChannelID:      channelID,
				EventMessageID: eventMessageID,
				Participants:   updatedParticipants,
			},
		}, nil
	})
}

// CheckAllScoresSubmittedResult is a custom struct to return data from CheckAllScoresSubmitted.
type CheckAllScoresSubmittedResult struct {
	AllScoresSubmitted bool
	PayloadData        interface{} // Will hold AllScoresSubmittedPayloadV1 or ScoresPartiallySubmittedPayloadV1 data
}

// CheckAllScoresSubmitted checks if all participants (or teams) in the round have submitted scores.
// Returns OperationResult containing either:
// - ScoresPartiallySubmittedPayloadV1 (some participants/teams missing scores)
// - AllScoresSubmittedPayloadV1 (all participants/teams have scores)
func (s *RoundService) CheckAllScoresSubmitted(
	ctx context.Context,
	payload roundevents.ParticipantScoreUpdatedPayloadV1,
) (results.OperationResult, error) {

	return s.withTelemetry(ctx, "CheckAllScoresSubmitted", payload.RoundID, func(ctx context.Context) (results.OperationResult, error) {

		var participantsWithScore []roundtypes.Participant
		var missingParticipants []roundtypes.Participant

		// 1. Split participants by score state
		for _, p := range payload.Participants {
			if p.Score != nil || p.Response == roundtypes.ResponseDecline {
				participantsWithScore = append(participantsWithScore, p)
			} else {
				missingParticipants = append(missingParticipants, p)
			}
		}

		// 2. Build teams from participants (TeamID is canonical)
		teamsByID := make(map[uuid.UUID]*roundtypes.NormalizedTeam)

		for _, p := range participantsWithScore {
			if p.TeamID == uuid.Nil || p.Score == nil {
				continue
			}

			team, ok := teamsByID[p.TeamID]
			if !ok {
				team = &roundtypes.NormalizedTeam{
					TeamID:  p.TeamID,
					Members: []roundtypes.TeamMember{},
				}
				teamsByID[p.TeamID] = team
			}

			team.Total += int(*p.Score)
			team.Members = append(team.Members, roundtypes.TeamMember{
				UserID:  &p.UserID,
				RawName: string(p.UserID),
			})
		}

		teamsWithScore := make([]roundtypes.NormalizedTeam, 0, len(teamsByID))
		for _, team := range teamsByID {
			teamsWithScore = append(teamsWithScore, *team)
		}

		// 3. Partial submission
		if len(missingParticipants) > 0 {
			return results.OperationResult{
				Success: &roundevents.ScoresPartiallySubmittedPayloadV1{
					GuildID:        payload.GuildID,
					RoundID:        payload.RoundID,
					UserID:         payload.UserID,
					Score:          payload.Score,
					EventMessageID: payload.EventMessageID,
					Participants:   participantsWithScore,
					Teams:          teamsWithScore,
				},
			}, nil
		}

		// 4. All scores submitted → fetch round ONCE
		round, err := s.repo.GetRound(ctx, payload.GuildID, payload.RoundID)
		if err != nil {
			return results.OperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
					GuildID: payload.GuildID,
					RoundID: payload.RoundID,
					Error:   fmt.Sprintf("failed to fetch round: %v", err),
				},
			}, nil
		}

		round.Participants = participantsWithScore
		round.Teams = teamsWithScore

		return results.OperationResult{
			Success: &roundevents.AllScoresSubmittedPayloadV1{
				GuildID:        payload.GuildID,
				RoundID:        payload.RoundID,
				EventMessageID: payload.EventMessageID,
				RoundData:      *round,
				Participants:   participantsWithScore,
				Teams:          teamsWithScore,
			},
		}, nil
	})
}

// CheckIfAllScoresSubmitted checks if all participants in the round have submitted scores.
// func (s *RoundService) checkIfAllScoresSubmitted(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (bool, error) {
// 	participants, err := s.repo.GetParticipants(ctx, guildID, roundID)
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
