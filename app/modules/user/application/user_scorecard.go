package userservice

import (
	"context"
	"fmt"
	"strings"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/uptrace/bun"
)

// MatchParsedScorecard matches player names from a scorecard to users.
func (s *UserService) MatchParsedScorecard(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
	playerNames []string,
) (results.OperationResult[*MatchResult, error], error) {

	matchOp := func(ctx context.Context, db bun.IDB) (results.OperationResult[*MatchResult, error], error) {
		return s.executeMatchParsedScorecard(ctx, db, guildID, userID, playerNames)
	}

	result, err := withTelemetry(s, ctx, "MatchParsedScorecard", userID, func(ctx context.Context) (results.OperationResult[*MatchResult, error], error) {
		return matchOp(ctx, s.db)
	})

	if err != nil {
		return results.OperationResult[*MatchResult, error]{}, fmt.Errorf("MatchParsedScorecard failed: %w", err)
	}

	return result, nil
}

func (s *UserService) executeMatchParsedScorecard(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	_ sharedtypes.DiscordID,
	playerNames []string,
) (results.OperationResult[*MatchResult, error], error) {
	const (
		maxPlayers  = 512
		maxNameRune = 128
	)

	if len(playerNames) > maxPlayers {
		return results.FailureResult[*MatchResult](fmt.Errorf("too many players: %d", len(playerNames))), nil
	}

	// 1. Normalize and collect unique names to look up
	uniqueNames := make(map[string]string) // normalized -> raw (one representation)
	unmatchedRaw := make([]string, 0)

	for _, raw := range playerNames {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if len([]rune(name)) > maxNameRune {
			unmatchedRaw = append(unmatchedRaw, name)
			continue
		}
		uniqueNames[strings.ToLower(name)] = raw
	}

	if len(uniqueNames) == 0 {
		return results.SuccessResult[*MatchResult, error](&MatchResult{
			Mappings:  []userevents.UDiscConfirmedMappingV1{},
			Unmatched: unmatchedRaw,
		}), nil
	}

	normList := make([]string, 0, len(uniqueNames))
	for norm := range uniqueNames {
		normList = append(normList, norm)
	}

	// 2. Batch lookup by Username
	byUsername, err := s.repo.GetUsersByUDiscUsernames(ctx, db, guildID, normList)
	if err != nil {
		return results.OperationResult[*MatchResult, error]{}, fmt.Errorf("batch username lookup failed: %w", err)
	}

	matchedUsers := make(map[string]sharedtypes.DiscordID)
	for _, u := range byUsername {
		if u.User.UDiscUsername != nil {
			matchedUsers[strings.ToLower(*u.User.UDiscUsername)] = u.User.GetUserID()
		}
	}

	// 3. Batch lookup by Name for names still unmatched
	stillMissing := make([]string, 0)
	for _, norm := range normList {
		if _, ok := matchedUsers[norm]; !ok {
			stillMissing = append(stillMissing, norm)
		}
	}

	if len(stillMissing) > 0 {
		byName, err := s.repo.GetUsersByUDiscNames(ctx, db, guildID, stillMissing)
		if err != nil {
			return results.OperationResult[*MatchResult, error]{}, fmt.Errorf("batch name lookup failed: %w", err)
		}
		for _, u := range byName {
			if u.User.UDiscName != nil {
				matchedUsers[strings.ToLower(*u.User.UDiscName)] = u.User.GetUserID()
			}
		}
	}

	// 4. Build output result map
	matchResult := &MatchResult{
		Mappings:  []userevents.UDiscConfirmedMappingV1{},
		Unmatched: unmatchedRaw,
	}

	// Keep track of which names we've already matched to avoid duplicates
	processedNorms := make(map[string]bool)

	for _, raw := range playerNames {
		name := strings.TrimSpace(raw)
		norm := strings.ToLower(name)
		if norm == "" || len([]rune(name)) > maxNameRune {
			continue
		}

		if userID, ok := matchedUsers[norm]; ok {
			matchResult.Mappings = append(matchResult.Mappings, userevents.UDiscConfirmedMappingV1{
				PlayerName:    raw,
				DiscordUserID: userID,
			})
		} else if !processedNorms[norm] {
			matchResult.Unmatched = append(matchResult.Unmatched, raw)
			processedNorms[norm] = true
		}
	}

	return results.SuccessResult[*MatchResult, error](matchResult), nil
}
