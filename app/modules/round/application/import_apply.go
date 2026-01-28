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
	"github.com/uptrace/bun"
)

// ApplyImportedScores applies imported scores to round participants in the database.
// Returns ImportScoresAppliedPayloadV1 for singles, or DoublesScoresReadyPayloadV1 for doubles.
func (s *RoundService) ApplyImportedScores(
	ctx context.Context,
	req roundtypes.ImportApplyScoresInput,
) (ApplyImportedScoresResult, error) {
	return withTelemetry(s, ctx, "ApplyImportedScores", req.RoundID, func(ctx context.Context) (ApplyImportedScoresResult, error) {
		return s.executeApplyImportedScores(ctx, req)
	})
}

func (s *RoundService) executeApplyImportedScores(
	ctx context.Context,
	req roundtypes.ImportApplyScoresInput,
) (ApplyImportedScoresResult, error) {
	return runInTx(s, ctx, func(ctx context.Context, tx bun.IDB) (ApplyImportedScoresResult, error) {
		if len(req.Scores) == 0 {
			return results.OperationResult[*roundtypes.ImportApplyScoresResult, error]{}, nil
		}

		// --- Singles vs Teams ---
		round, err := s.repo.GetRound(ctx, tx, req.GuildID, req.RoundID)
		if err != nil {
			return results.FailureResult[*roundtypes.ImportApplyScoresResult, error](fmt.Errorf("failed to get round: %w", err)), nil
		}

		if len(round.Teams) > 0 {
			return s.applyTeamScores(ctx, tx, req, round)
		}

		return s.applySinglesScores(ctx, tx, req, round)
	})
}

// applySinglesScores applies scores for singles mode imports.
func (s *RoundService) applySinglesScores(
	ctx context.Context,
	tx bun.IDB,
	req roundtypes.ImportApplyScoresInput,
	round *roundtypes.Round,
) (ApplyImportedScoresResult, error) {
	// Get existing participants to check for duplicates and update in place
	existingParticipants, err := s.repo.GetParticipants(ctx, tx, req.GuildID, req.RoundID)
	if err != nil {
		return results.FailureResult[*roundtypes.ImportApplyScoresResult, error](fmt.Errorf("failed to get existing participants: %w", err)), nil
	}

	// Build map of existing participants by UserID for deduplication
	existingMap := make(map[sharedtypes.DiscordID]int)
	for i, p := range existingParticipants {
		if p.UserID != "" {
			existingMap[p.UserID] = i
		}
	}

	updatedCount := 0
	for _, scoreInfo := range req.Scores {
		// Do not create guest participants for singles; only add matched guild_membership users.
		// If UserID is empty, this is an unmatched/guest player - skip entirely for singles.
		if scoreInfo.UserID == "" {
			s.logger.DebugContext(ctx, "Skipping guest user in singles import (no guest participants for singles)",
				attr.String("raw_name", scoreInfo.RawName),
			)
			continue
		}

		// UserID is present, meaning resolveUserID found a match (user has guild_membership)
		score := sharedtypes.Score(scoreInfo.Score)
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
		return results.FailureResult[*roundtypes.ImportApplyScoresResult, error](errors.New("no scores were successfully applied")), nil
	}

	// Persist participants using existing repository method for consistent team derivation
	updates := []roundtypes.RoundUpdate{{
		RoundID:      req.RoundID,
		Participants: existingParticipants,
	}}

	if err := s.repo.UpdateRoundsAndParticipants(ctx, tx, req.GuildID, updates); err != nil {
		return results.FailureResult[*roundtypes.ImportApplyScoresResult, error](fmt.Errorf("failed to persist participants: %w", err)), nil
	}

	return results.SuccessResult[*roundtypes.ImportApplyScoresResult, error](&roundtypes.ImportApplyScoresResult{
		GuildID:        req.GuildID,
		RoundID:        req.RoundID,
		ImportID:       req.ImportID,
		Participants:   existingParticipants,
		EventMessageID: round.EventMessageID,
		Timestamp:      time.Now().UTC(),
	}), nil
}

// --- New: applyTeamScores handles doubles/multi-player teams ---
func (s *RoundService) applyTeamScores(
	ctx context.Context,
	tx bun.IDB,
	req roundtypes.ImportApplyScoresInput,
	round *roundtypes.Round,
) (ApplyImportedScoresResult, error) {
	existingParticipants, err := s.repo.GetParticipants(ctx, tx, req.GuildID, req.RoundID)
	if err != nil {
		return results.FailureResult[*roundtypes.ImportApplyScoresResult, error](fmt.Errorf("failed to get participants: %w", err)), nil
	}

	hasGroups, err := s.repo.RoundHasGroups(ctx, tx, req.RoundID)
	if err != nil {
		return results.FailureResult[*roundtypes.ImportApplyScoresResult, error](fmt.Errorf("failed checking round groups: %w", err)), nil
	}

	if !hasGroups {
		participantsToGroup := make([]roundtypes.Participant, len(req.Scores))
		for i, sc := range req.Scores {
			score := sharedtypes.Score(sc.Score)
			participantsToGroup[i] = roundtypes.Participant{
				UserID:   sc.UserID,
				Score:    &score,
				Response: roundtypes.ResponseAccept,
				TeamID:   sc.TeamID,
				RawName:  sc.RawName,
			}
		}

		if err := s.repo.CreateRoundGroups(ctx, tx, req.RoundID, participantsToGroup); err != nil {
			return results.FailureResult[*roundtypes.ImportApplyScoresResult, error](fmt.Errorf("failed to create round groups: %w", err)), nil
		}
	}

	// Build map of existing participants by UserID
	existingMap := make(map[sharedtypes.DiscordID]int)
	for i, p := range existingParticipants {
		existingMap[p.UserID] = i
	}

	// Update existing participants and add new ones from import
	for _, sc := range req.Scores {
		// Guest users (empty UserID) are always added as new participants
		if sc.UserID == "" {
			score := sharedtypes.Score(sc.Score)
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
			score := sharedtypes.Score(sc.Score)
			existingParticipants[idx].Score = &score
			existingParticipants[idx].Response = roundtypes.ResponseAccept
			existingParticipants[idx].TeamID = sc.TeamID
		} else {
			// Add new participant (user wasn't RSVP'd but is in the scorecard)
			score := sharedtypes.Score(sc.Score)
			existingParticipants = append(existingParticipants, roundtypes.Participant{
				UserID:   sc.UserID,
				Score:    &score,
				Response: roundtypes.ResponseAccept,
				TeamID:   sc.TeamID,
			})
		}
	}

	updates := []roundtypes.RoundUpdate{{
		RoundID:      req.RoundID,
		Participants: existingParticipants,
	}}

	if err := s.repo.UpdateRoundsAndParticipants(ctx, tx, req.GuildID, updates); err != nil {
		return results.FailureResult[*roundtypes.ImportApplyScoresResult, error](fmt.Errorf("failed to update rounds and participants: %w", err)), nil
	}

	return results.SuccessResult[*roundtypes.ImportApplyScoresResult, error](&roundtypes.ImportApplyScoresResult{
		GuildID:        req.GuildID,
		RoundID:        req.RoundID,
		ImportID:       req.ImportID,
		Participants:   existingParticipants,
		EventMessageID: round.EventMessageID,
		Timestamp:      time.Now().UTC(),
	}), nil
}
