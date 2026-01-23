package roundservice

import (
	"context"
	"strings"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
)

// domain-friendly format and detects the Round Mode (Singles vs Doubles).
func (s *RoundService) NormalizeParsedScorecard(ctx context.Context, data *roundtypes.ParsedScorecard, meta roundtypes.Metadata) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "NormalizeParsedScorecard", meta.RoundID, func(ctx context.Context) (results.OperationResult, error) {
		s.logger.InfoContext(ctx, "Normalizing parsed scorecard",
			attr.String("import_id", meta.ImportID),
			attr.String("round_id", meta.RoundID.String()),
		)

		if data == nil {
			return results.OperationResult{
				Failure: importFailure(roundevents.ScorecardUploadedPayloadV1{
					GuildID: meta.GuildID, RoundID: meta.RoundID, ImportID: meta.ImportID, UserID: meta.UserID, ChannelID: meta.ChannelID,
				}, errCodeUnsupported, "parsed scorecard data is nil"),
			}, nil
		}

		// 1. Initialize the Normalized structure defined in your roundtypes
		normalizedRound := roundtypes.NormalizedScorecard{
			ID:        uuid.NewString(), // Internal ID for this normalization instance
			RoundID:   meta.RoundID,
			GuildID:   meta.GuildID,
			ImportID:  meta.ImportID,
			Mode:      data.Mode, // Uses the Mode (SINGLES/DOUBLES) from the parser
			ParScores: cloneInts(data.ParScores),
			CreatedAt: time.Now().UTC(),
		}

		// 2. Map PlayerScoreRow to NormalizedPlayer or NormalizedTeam
		if data.Mode == sharedtypes.RoundModeDoubles {
			for _, p := range data.PlayerScores {

				teamID := uuid.New()

				team := roundtypes.NormalizedTeam{
					TeamID:     teamID,
					Total:      p.Total,
					HoleScores: cloneInts(p.HoleScores),
				}

				for _, name := range p.TeamNames {
					team.Members = append(team.Members, roundtypes.TeamMember{
						RawName: strings.TrimSpace(name),
					})
				}

				if len(team.Members) == 0 && p.PlayerName != "" {
					team.Members = append(team.Members, roundtypes.TeamMember{
						RawName: strings.TrimSpace(p.PlayerName),
					})
				}

				normalizedRound.Teams = append(normalizedRound.Teams, team)
			}
		} else {
			// Singles Mode (no TeamID)
			for _, p := range data.PlayerScores {
				normalizedRound.Players = append(normalizedRound.Players, roundtypes.NormalizedPlayer{
					DisplayName: strings.TrimSpace(p.PlayerName),
					Total:       p.Total,
					HoleScores:  cloneInts(p.HoleScores),
				})
			}
		}

		// 3. Wrap into the Event Payload
		successPayload := &roundevents.ScorecardNormalizedPayloadV1{
			ImportID:       meta.ImportID,
			GuildID:        meta.GuildID,
			RoundID:        meta.RoundID,
			UserID:         meta.UserID,
			ChannelID:      meta.ChannelID,
			Normalized:     normalizedRound, // The nested struct from roundtypes
			EventMessageID: meta.EventMessageID,
			Timestamp:      time.Now().UTC(),
		}

		s.logger.InfoContext(ctx, "Normalization complete",
			attr.String("mode", string(data.Mode)),
			attr.Int("player_count", len(data.PlayerScores)),
		)

		return results.OperationResult{
			Success: successPayload,
		}, nil
	})
}

// IngestNormalizedScorecard performs user matching and prepares the scores for final application.
func (s *RoundService) IngestNormalizedScorecard(
	ctx context.Context,
	payload roundevents.ScorecardNormalizedPayloadV1,
) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "IngestNormalizedScorecard", payload.RoundID, func(ctx context.Context) (results.OperationResult, error) {
		s.logger.InfoContext(ctx, "Ingesting normalized scorecard",
			attr.String("import_id", payload.ImportID),
			attr.String("mode", string(payload.Normalized.Mode)),
			attr.Int("teams_count", len(payload.Normalized.Teams)),
			attr.Int("players_count", len(payload.Normalized.Players)),
		)

		var finalScores []sharedtypes.ScoreInfo
		matchedCount := 0
		unmatchedPlayers := make([]string, 0)
		groupsToCreate := []roundtypes.Participant{}

		// --- Handle Mode: Doubles / Teams ---
		if payload.Normalized.Mode != sharedtypes.RoundModeSingles {
			for _, team := range payload.Normalized.Teams {
				teamMatched := false
				for _, member := range team.Members {
					normalizedName := normalizeName(member.RawName)
					discordID := s.resolveUserID(ctx, payload.GuildID, normalizedName)

					// Prepare participant for DB group creation
					groupsToCreate = append(groupsToCreate, roundtypes.Participant{
						UserID: discordID,
					})

					if discordID != "" {
						finalScores = append(finalScores, sharedtypes.ScoreInfo{
							UserID: discordID,
							Score:  sharedtypes.Score(team.Total),
						})
						matchedCount++
						teamMatched = true
					} else {
						unmatchedPlayers = append(unmatchedPlayers, member.RawName)
					}
				}

				// Optional logging if team has no matched members
				if !teamMatched && len(team.Members) > 0 {
					s.logger.WarnContext(ctx, "No members matched for team", attr.UUIDValue("team_id", team.TeamID))
				}
			}

			// --- Create RoundGroups for this round ---
			if len(groupsToCreate) > 0 {
				hasGroups, err := s.repo.RoundHasGroups(ctx, payload.RoundID)
				if err != nil {
					return results.OperationResult{
						Failure: importFailureFromNormalized(payload, "DB_ERROR", "failed checking existing round groups"),
					}, err
				}

				if !hasGroups {
					if err := s.repo.CreateRoundGroups(ctx, payload.RoundID, groupsToCreate); err != nil {
						return results.OperationResult{
							Failure: importFailureFromNormalized(payload, "DB_ERROR", "failed creating round groups"),
						}, err
					}
				}
			}

		} else {
			// --- Singles mode ---
			for _, p := range payload.Normalized.Players {
				normalizedName := normalizeName(p.DisplayName)
				discordID := s.resolveUserID(ctx, payload.GuildID, normalizedName)
				if discordID == "" {
					unmatchedPlayers = append(unmatchedPlayers, p.DisplayName)
					continue
				}
				finalScores = append(finalScores, sharedtypes.ScoreInfo{
					UserID: discordID,
					Score:  sharedtypes.Score(p.Total),
				})
				matchedCount++
			}
		}

		if len(finalScores) == 0 {
			return results.OperationResult{
				Failure: importFailureFromNormalized(payload, "NO_MATCHES", "no valid player scores matched"),
			}, nil
		}

		// --- Return ImportCompletedPayloadV1 ---
		return results.OperationResult{
			Success: &roundevents.ImportCompletedPayloadV1{
				ImportID:         payload.ImportID,
				GuildID:          payload.GuildID,
				RoundID:          payload.RoundID,
				UserID:           payload.UserID,
				ChannelID:        payload.ChannelID,
				ScoresIngested:   len(finalScores),
				MatchedPlayers:   matchedCount,
				UnmatchedPlayers: len(unmatchedPlayers),
				SkippedPlayers:   unmatchedPlayers,
				Scores:           finalScores,
				RoundMode:        payload.Normalized.Mode,
				EventMessageID:   payload.EventMessageID,
				Timestamp:        time.Now().UTC(),
			},
		}, nil
	})
}

// Helper to extract strings from TeamMember structs
func extractRawNames(members []roundtypes.TeamMember) []string {
	names := make([]string, len(members))
	for i, m := range members {
		names[i] = m.RawName
	}
	return names
}

// ingestDoublesScorecard handles doubles ingestion (team matching).
func (s *RoundService) ingestDoublesScorecard(ctx context.Context, payload roundevents.ScorecardNormalizedPayloadV1, round *roundtypes.Round) (results.OperationResult, error) {
	var scores []sharedtypes.ScoreInfo
	var matched []roundtypes.MatchedPlayer
	var unmatched []string

	for _, team := range payload.Normalized.Teams {
		teamMatched := false
		teamScores := make([]sharedtypes.ScoreInfo, 0, len(team.Members))
		for _, member := range team.Members {
			name := normalizeName(member.RawName)
			if name == "" {
				unmatched = append(unmatched, member.RawName)
				continue
			}
			userID := s.resolveUserID(ctx, payload.GuildID, name)
			if userID == "" {
				unmatched = append(unmatched, member.RawName)
				continue
			}
			teamScores = append(teamScores, sharedtypes.ScoreInfo{UserID: userID, Score: sharedtypes.Score(team.Total), TagNumber: nil})
			matched = append(matched, roundtypes.MatchedPlayer{DiscordID: userID, UDiscName: member.RawName, Score: team.Total})
			teamMatched = true
		}
		if teamMatched {
			scores = append(scores, teamScores...)
		} else {
			var memberNames []string
			for _, m := range team.Members {
				memberNames = append(memberNames, m.RawName)
			}
			unmatched = append(unmatched, strings.Join(memberNames, " + "))
		}
	}

	if len(scores) == 0 {
		return results.OperationResult{Failure: importFailureFromNormalized(payload, "NO_MATCHES", "no team members matched")}, nil
	}

	return results.OperationResult{Success: &roundevents.ImportCompletedPayloadV1{
		ImportID:           payload.ImportID,
		GuildID:            payload.GuildID,
		RoundID:            payload.RoundID,
		UserID:             payload.UserID,
		ChannelID:          payload.ChannelID,
		ScoresIngested:     len(scores),
		MatchedPlayers:     len(matched),
		UnmatchedPlayers:   len(unmatched),
		MatchedPlayersList: matched,
		SkippedPlayers:     unmatched,
		Scores:             scores,
		EventMessageID:     payload.EventMessageID,
		Timestamp:          time.Now().UTC(),
		RoundMode:          sharedtypes.RoundModeDoubles,
	}}, nil
}

// resolveUserID attempts to match a normalized UDisc name to a Discord user ID.
func (s *RoundService) resolveUserID(ctx context.Context, guildID sharedtypes.GuildID, normalizedName string) sharedtypes.DiscordID {
	// Validation: Warn if called with non-normalized name
	if normalizedName != normalizeName(normalizedName) {
		s.logger.WarnContext(ctx, "resolveUserID called with non-normalized name",
			attr.String("input", normalizedName))
	}

	if s.userLookup == nil {
		return ""
	}

	// 1. Try exact username match
	identity, err := s.userLookup.FindByNormalizedUDiscUsername(ctx, guildID, normalizedName)
	if err == nil && identity != nil {
		return identity.UserID
	}

	// 2. Try exact display name match
	identity, err = s.userLookup.FindByNormalizedUDiscDisplayName(ctx, guildID, normalizedName)
	if err == nil && identity != nil {
		return identity.UserID
	}

	// 3. Try fuzzy match (ONLY if exactly 1 match)
	identities, err := s.userLookup.FindByPartialUDiscName(ctx, guildID, normalizedName)
	if err == nil && len(identities) == 1 {
		s.logger.InfoContext(ctx, "Fuzzy match found",
			attr.String("search_term", normalizedName),
			attr.String("matched_user_id", string(identities[0].UserID)))
		return identities[0].UserID
	}

	if len(identities) > 1 {
		s.logger.WarnContext(ctx, "Ambiguous fuzzy match, skipping",
			attr.String("search_term", normalizedName),
			attr.Int("match_count", len(identities)))
	}

	return ""
}
