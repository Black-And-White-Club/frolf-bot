package roundservice

import (
	"context"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
)

// ApplyImportedScores applies imported scores to round participants in the database.
// Returns ImportScoresAppliedPayloadV1 for singles, or DoublesScoresReadyPayloadV1 for doubles.
func (s *RoundService) ApplyImportedScores(
	ctx context.Context,
	payload roundevents.ImportCompletedPayloadV1,
) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "ApplyImportedScores", payload.RoundID, func(ctx context.Context) (results.OperationResult, error) {
		if len(payload.Scores) == 0 {
			return results.OperationResult{}, nil
		}

		// --- Singles vs Teams ---
		if payload.RoundMode == sharedtypes.RoundModeDoubles || payload.RoundMode == sharedtypes.RoundModeTriples {
			return s.applyTeamScores(ctx, payload)
		}

		return s.applySinglesScores(ctx, payload)
	})
}

// applySinglesScores applies scores for singles mode imports.
// For singles: only add participants when a matched user (with guild_membership) is found.
// Do not create guest participants for singles; only add matched guild_membership users.
func (s *RoundService) applySinglesScores(
	ctx context.Context,
	payload roundevents.ImportCompletedPayloadV1,
) (results.OperationResult, error) {
	// Get existing participants to check for duplicates and update in place
	existingParticipants, err := s.repo.GetParticipants(ctx, payload.GuildID, payload.RoundID)
	if err != nil {
		return results.OperationResult{
			Failure: &roundevents.ImportFailedPayloadV1{
				GuildID:   payload.GuildID,
				RoundID:   payload.RoundID,
				ImportID:  payload.ImportID,
				UserID:    payload.UserID,
				ChannelID: payload.ChannelID,
				Error:     fmt.Sprintf("failed to get existing participants: %v", err),
				ErrorCode: "DB_ERROR",
				Timestamp: time.Now().UTC(),
			},
		}, nil
	}

	// Build map of existing participants by UserID for deduplication
	existingMap := make(map[sharedtypes.DiscordID]int)
	for i, p := range existingParticipants {
		if p.UserID != "" {
			existingMap[p.UserID] = i
		}
	}

	updatedCount := 0
	for _, scoreInfo := range payload.Scores {
		// Do not create guest participants for singles; only add matched guild_membership users.
		// If UserID is empty, this is an unmatched/guest player - skip entirely for singles.
		if scoreInfo.UserID == "" {
			s.logger.DebugContext(ctx, "Skipping guest user in singles import (no guest participants for singles)",
				attr.String("raw_name", scoreInfo.RawName),
			)
			continue
		}

		// UserID is present, meaning resolveUserID found a match (user has guild_membership)
		score := scoreInfo.Score
		if idx, exists := existingMap[scoreInfo.UserID]; exists {
			// Update existing participant's score
			existingParticipants[idx].Score = &score
			existingParticipants[idx].Response = roundtypes.ResponseAccept
		} else {
			// Add new participant - user wasn't RSVP'd but is in the scorecard and has guild_membership
			existingParticipants = append(existingParticipants, roundtypes.Participant{
				UserID:   scoreInfo.UserID,
				Score:    &score,
				Response: roundtypes.ResponseAccept,
			})
			// Update map to prevent duplicates within same import
			existingMap[scoreInfo.UserID] = len(existingParticipants) - 1
		}
		updatedCount++
	}

	if updatedCount == 0 {
		return results.OperationResult{
			Failure: &roundevents.ImportFailedPayloadV1{
				GuildID:   payload.GuildID,
				RoundID:   payload.RoundID,
				ImportID:  payload.ImportID,
				UserID:    payload.UserID,
				ChannelID: payload.ChannelID,
				Error:     "no scores were successfully applied",
				ErrorCode: "NO_UPDATES",
				Timestamp: time.Now().UTC(),
			},
		}, nil
	}

	// Persist participants using existing repository method for consistent team derivation
	updates := []roundtypes.RoundUpdate{{
		RoundID:      payload.RoundID,
		Participants: existingParticipants,
	}}

	if err := s.repo.UpdateRoundsAndParticipants(ctx, payload.GuildID, updates); err != nil {
		return results.OperationResult{
			Failure: &roundevents.ImportFailedPayloadV1{
				GuildID:   payload.GuildID,
				RoundID:   payload.RoundID,
				ImportID:  payload.ImportID,
				UserID:    payload.UserID,
				ChannelID: payload.ChannelID,
				Error:     fmt.Sprintf("failed to persist participants: %v", err),
				ErrorCode: "DB_ERROR",
				Timestamp: time.Now().UTC(),
			},
		}, nil
	}

	round, err := s.repo.GetRound(ctx, payload.GuildID, payload.RoundID)
	if err != nil {
		return results.OperationResult{
			Failure: &roundevents.RoundErrorPayloadV1{
				GuildID: payload.GuildID,
				RoundID: payload.RoundID,
				Error:   fmt.Sprintf("failed to get round: %v", err),
			},
		}, nil
	}

	return results.OperationResult{
		Success: &roundevents.ImportScoresAppliedPayloadV1{
			GuildID:        payload.GuildID,
			RoundID:        payload.RoundID,
			ImportID:       payload.ImportID,
			Participants:   existingParticipants,
			EventMessageID: round.EventMessageID,
			Timestamp:      time.Now().UTC(),
		},
	}, nil
}

// --- New: applyTeamScores handles doubles/multi-player teams ---
func (s *RoundService) applyTeamScores(
	ctx context.Context,
	payload roundevents.ImportCompletedPayloadV1,
) (results.OperationResult, error) {
	existingParticipants, err := s.repo.GetParticipants(ctx, payload.GuildID, payload.RoundID)
	if err != nil {
		return results.OperationResult{Failure: &roundevents.RoundErrorPayloadV1{Error: err.Error()}}, nil
	}

	hasGroups, err := s.repo.RoundHasGroups(ctx, payload.RoundID)
	if err != nil {
		return results.OperationResult{Failure: &roundevents.RoundErrorPayloadV1{Error: err.Error()}}, nil
	}

	if !hasGroups {
		if err := s.repo.CreateRoundGroups(ctx, payload.RoundID, s.mapScoresToParticipants(payload.Scores)); err != nil {
			return results.OperationResult{Failure: &roundevents.RoundErrorPayloadV1{Error: err.Error()}}, nil
		}
	}

	// Build map of existing participants by UserID
	existingMap := make(map[sharedtypes.DiscordID]int)
	for i, p := range existingParticipants {
		existingMap[p.UserID] = i
	}

	// Update existing participants and add new ones from import
	for _, sc := range payload.Scores {
		// Guest users (empty UserID) are always added as new participants
		if sc.UserID == "" {
			score := sc.Score
			existingParticipants = append(existingParticipants, roundtypes.Participant{
				UserID:   "",
				Score:    &score,
				Response: roundtypes.ResponseAccept,
				TeamID:   sc.TeamID,
				RawName:  sc.RawName,
			})
			continue
		}

		if idx, exists := existingMap[sc.UserID]; exists {
			// Update existing participant
			score := sc.Score
			existingParticipants[idx].Score = &score
			existingParticipants[idx].Response = roundtypes.ResponseAccept
			existingParticipants[idx].TeamID = sc.TeamID
		} else {
			// Add new participant (user wasn't RSVP'd but is in the scorecard)
			score := sc.Score
			existingParticipants = append(existingParticipants, roundtypes.Participant{
				UserID:   sc.UserID,
				Score:    &score,
				Response: roundtypes.ResponseAccept,
				TeamID:   sc.TeamID,
			})
		}
	}

	updates := []roundtypes.RoundUpdate{{
		RoundID:      payload.RoundID,
		Participants: existingParticipants,
	}}

	if err := s.repo.UpdateRoundsAndParticipants(ctx, payload.GuildID, updates); err != nil {
		return results.OperationResult{Failure: &roundevents.RoundErrorPayloadV1{Error: err.Error()}}, nil
	}

	round, err := s.repo.GetRound(ctx, payload.GuildID, payload.RoundID)
	if err != nil {
		s.logger.WarnContext(ctx, "Failed to get round for event message ID", attr.Error(err))
	}

	// Return standard applied payload so handler can trigger updates
	return results.OperationResult{
		Success: &roundevents.ImportScoresAppliedPayloadV1{
			GuildID:        payload.GuildID,
			RoundID:        payload.RoundID,
			ImportID:       payload.ImportID,
			Participants:   existingParticipants,
			EventMessageID: round.EventMessageID,
			Timestamp:      time.Now().UTC(),
		},
	}, nil
}

// --- Helper: mapScoresToParticipants converts ScoreInfo -> Participant ---
func (s *RoundService) mapScoresToParticipants(scores []sharedtypes.ScoreInfo) []roundtypes.Participant {
	participants := make([]roundtypes.Participant, len(scores))
	for i, sc := range scores {
		score := sc.Score
		participants[i] = roundtypes.Participant{
			UserID:   sc.UserID,
			Score:    &score,
			Response: roundtypes.ResponseAccept,
			TeamID:   sc.TeamID,
			RawName:  sc.RawName,
		}
	}
	return participants
}
