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
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
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
		source := normalizeImportSource(req.Source)
		importInputKind := "unknown"
		importFileExt := "unknown"
		roundState := "unknown"

		if len(req.Scores) == 0 {
			return results.OperationResult[*roundtypes.ImportApplyScoresResult, error]{}, nil
		}

		// --- Singles vs Teams ---
		round, err := s.repo.GetRound(ctx, tx, req.GuildID, req.RoundID)
		if err != nil {
			failureErr := fmt.Errorf("failed to get round: %w", err)
			s.recordImportFailure(ctx, source, importInputKind, importFileExt, roundState, failureErr)
			return results.FailureResult[*roundtypes.ImportApplyScoresResult, error](failureErr), nil
		}
		roundState = roundStateValue(round)
		importInputKind = inputKindFromRound(round)
		importFileExt = fileExt(round.FileName, "", round.UDiscURL)

		applyStart := time.Now()
		defer func() {
			s.recordImportPhaseDuration(ctx, importPhaseApply, source, importInputKind, importFileExt, time.Since(applyStart))
		}()

		var result ApplyImportedScoresResult
		if len(round.Teams) > 0 {
			result, err = s.applyTeamScores(ctx, tx, req, round)
		} else {
			result, err = s.applySinglesScores(ctx, tx, req, round)
		}
		if err != nil {
			s.recordImportFailure(ctx, source, importInputKind, importFileExt, roundState, err)
			return result, err
		}
		if result.Failure != nil {
			s.recordImportFailure(ctx, source, importInputKind, importFileExt, roundState, *result.Failure)
			return result, nil
		}
		if req.ImportID == "" {
			return result, nil
		}

		if err := s.repo.UpdateImportStatus(
			ctx,
			tx,
			req.GuildID,
			req.RoundID,
			req.ImportID,
			string(rounddb.ImportStatusCompleted),
			"",
			"",
		); err != nil {
			failureErr := fmt.Errorf("failed to mark import as completed: %w", err)
			s.recordImportFailure(ctx, source, importInputKind, importFileExt, roundState, failureErr)
			return results.FailureResult[*roundtypes.ImportApplyScoresResult, error](
				failureErr,
			), nil
		}
		s.importerMetrics.RecordImportSuccess(ctx, source, importInputKind, importFileExt, roundState)

		return result, nil
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
	if req.OverwriteExistingScores {
		return s.overwriteSinglesScores(ctx, tx, req, round, existingParticipants)
	}

	// Build map of existing participants by UserID for deduplication
	existingMap := make(map[sharedtypes.DiscordID]int)
	for i, p := range existingParticipants {
		if p.UserID != "" {
			existingMap[p.UserID] = i
		}
	}
	guestIndexByName := make(map[string]int)
	for i, p := range existingParticipants {
		if p.UserID != "" {
			continue
		}
		normalized := normalizeName(p.RawName)
		if normalized == "" {
			continue
		}
		guestIndexByName[normalized] = i
	}

	updatedCount := 0
	for _, scoreInfo := range req.Scores {
		if scoreInfo.UserID == "" {
			if !req.AllowGuestPlayers {
				s.logger.DebugContext(ctx, "Skipping guest user in singles import (guest participants disabled)",
					attr.String("raw_name", scoreInfo.RawName),
				)
				continue
			}

			normalizedGuestName := normalizeName(scoreInfo.RawName)
			if normalizedGuestName == "" {
				continue
			}

			score := sharedtypes.Score(scoreInfo.Score)
			if idx, exists := guestIndexByName[normalizedGuestName]; exists {
				existingParticipants[idx].Score = &score
				existingParticipants[idx].Response = roundtypes.ResponseAccept
				existingParticipants[idx].RawName = scoreInfo.RawName
				existingParticipants[idx].HoleScores = scoreInfo.HoleScores
				existingParticipants[idx].IsDNF = scoreInfo.IsDNF
			} else {
				existingParticipants = append(existingParticipants, roundtypes.Participant{
					UserID:     "",
					RawName:    scoreInfo.RawName,
					Score:      &score,
					Response:   roundtypes.ResponseAccept,
					HoleScores: scoreInfo.HoleScores,
					IsDNF:      scoreInfo.IsDNF,
				})
				guestIndexByName[normalizedGuestName] = len(existingParticipants) - 1
			}
			updatedCount++
			continue
		}

		// UserID is present, meaning resolveUserID found a match (user has guild_membership)
		score := sharedtypes.Score(scoreInfo.Score)
		if idx, exists := existingMap[scoreInfo.UserID]; exists {
			// Update existing participant's score
			existingParticipants[idx].Score = &score
			existingParticipants[idx].Response = roundtypes.ResponseAccept
			existingParticipants[idx].HoleScores = scoreInfo.HoleScores
			existingParticipants[idx].IsDNF = scoreInfo.IsDNF
		} else {
			// Add new participant - user wasn't RSVP'd but is in the scorecard and has guild_membership
			existingParticipants = append(existingParticipants, roundtypes.Participant{
				UserID:     scoreInfo.UserID,
				Score:      &score,
				Response:   roundtypes.ResponseAccept,
				HoleScores: scoreInfo.HoleScores,
				IsDNF:      scoreInfo.IsDNF,
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

	if len(req.ParScores) > 0 {
		round.ParScores = req.ParScores
		if _, err := s.repo.UpdateRound(ctx, tx, req.GuildID, req.RoundID, round); err != nil {
			return results.FailureResult[*roundtypes.ImportApplyScoresResult, error](fmt.Errorf("failed to update round par scores: %w", err)), nil
		}
	}

	return results.SuccessResult[*roundtypes.ImportApplyScoresResult, error](&roundtypes.ImportApplyScoresResult{
		GuildID:        req.GuildID,
		RoundID:        req.RoundID,
		ImportID:       req.ImportID,
		Participants:   existingParticipants,
		RoundData:      round,
		EventMessageID: round.EventMessageID,
		Timestamp:      time.Now().UTC(),
	}), nil
}

func (s *RoundService) overwriteSinglesScores(
	ctx context.Context,
	tx bun.IDB,
	req roundtypes.ImportApplyScoresInput,
	round *roundtypes.Round,
	existingParticipants []roundtypes.Participant,
) (ApplyImportedScoresResult, error) {
	existingByUserID := make(map[sharedtypes.DiscordID]roundtypes.Participant)
	for _, participant := range existingParticipants {
		if participant.UserID == "" {
			continue
		}
		existingByUserID[participant.UserID] = participant
	}

	participants := make([]roundtypes.Participant, 0, len(req.Scores))
	indexByKey := make(map[string]int)
	updatedCount := 0

	for _, scoreInfo := range req.Scores {
		if scoreInfo.UserID == "" && !req.AllowGuestPlayers {
			continue
		}

		score := sharedtypes.Score(scoreInfo.Score)
		next := roundtypes.Participant{
			UserID:     scoreInfo.UserID,
			RawName:    scoreInfo.RawName,
			Score:      &score,
			Response:   roundtypes.ResponseAccept,
			HoleScores: scoreInfo.HoleScores,
			IsDNF:      scoreInfo.IsDNF,
		}

		var key string
		if scoreInfo.UserID != "" {
			if existing, exists := existingByUserID[scoreInfo.UserID]; exists {
				next.TagNumber = existing.TagNumber
			}
			key = "user:" + string(scoreInfo.UserID)
		} else {
			normalizedGuestName := normalizeName(scoreInfo.RawName)
			if normalizedGuestName == "" {
				continue
			}
			key = "guest:" + normalizedGuestName
		}

		if idx, exists := indexByKey[key]; exists {
			participants[idx] = next
		} else {
			indexByKey[key] = len(participants)
			participants = append(participants, next)
		}
		updatedCount++
	}

	if updatedCount == 0 {
		return results.FailureResult[*roundtypes.ImportApplyScoresResult, error](errors.New("no scores were successfully applied")), nil
	}

	updates := []roundtypes.RoundUpdate{{
		RoundID:      req.RoundID,
		Participants: participants,
	}}

	if err := s.repo.UpdateRoundsAndParticipants(ctx, tx, req.GuildID, updates); err != nil {
		return results.FailureResult[*roundtypes.ImportApplyScoresResult, error](fmt.Errorf("failed to persist participants: %w", err)), nil
	}

	if len(req.ParScores) > 0 {
		round.ParScores = req.ParScores
		if _, err := s.repo.UpdateRound(ctx, tx, req.GuildID, req.RoundID, round); err != nil {
			return results.FailureResult[*roundtypes.ImportApplyScoresResult, error](fmt.Errorf("failed to update round par scores: %w", err)), nil
		}
	}

	return results.SuccessResult[*roundtypes.ImportApplyScoresResult, error](&roundtypes.ImportApplyScoresResult{
		GuildID:        req.GuildID,
		RoundID:        req.RoundID,
		ImportID:       req.ImportID,
		Participants:   participants,
		RoundData:      round,
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
				UserID:     sc.UserID,
				Score:      &score,
				Response:   roundtypes.ResponseAccept,
				TeamID:     sc.TeamID,
				RawName:    sc.RawName,
				IsDNF:      sc.IsDNF,
				HoleScores: sc.HoleScores,
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
				UserID:     "",
				Score:      &score,
				Response:   roundtypes.ResponseAccept,
				TeamID:     sc.TeamID,
				RawName:    sc.RawName,
				IsDNF:      sc.IsDNF,
				HoleScores: sc.HoleScores,
			})
			continue
		}

		if idx, exists := existingMap[sc.UserID]; exists {
			// Update existing participant
			score := sharedtypes.Score(sc.Score)
			existingParticipants[idx].Score = &score
			existingParticipants[idx].Response = roundtypes.ResponseAccept
			existingParticipants[idx].TeamID = sc.TeamID
			existingParticipants[idx].IsDNF = sc.IsDNF
			existingParticipants[idx].HoleScores = sc.HoleScores
		} else {
			// Add new participant (user wasn't RSVP'd but is in the scorecard)
			score := sharedtypes.Score(sc.Score)
			existingParticipants = append(existingParticipants, roundtypes.Participant{
				UserID:     sc.UserID,
				Score:      &score,
				Response:   roundtypes.ResponseAccept,
				TeamID:     sc.TeamID,
				IsDNF:      sc.IsDNF,
				HoleScores: sc.HoleScores,
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

	if len(req.ParScores) > 0 {
		round.ParScores = req.ParScores
		if _, err := s.repo.UpdateRound(ctx, tx, req.GuildID, req.RoundID, round); err != nil {
			return results.FailureResult[*roundtypes.ImportApplyScoresResult, error](fmt.Errorf("failed to update round par scores: %w", err)), nil
		}
	}

	return results.SuccessResult[*roundtypes.ImportApplyScoresResult, error](&roundtypes.ImportApplyScoresResult{
		GuildID:        req.GuildID,
		RoundID:        req.RoundID,
		ImportID:       req.ImportID,
		Participants:   existingParticipants,
		Teams:          round.Teams,
		RoundData:      round,
		EventMessageID: round.EventMessageID,
		Timestamp:      time.Now().UTC(),
	}), nil
}
