package roundservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// NormalizeParsedScorecard converts raw parsed data into a standard domain-friendly format.
func (s *RoundService) NormalizeParsedScorecard(ctx context.Context, data *roundtypes.ParsedScorecard, meta roundtypes.Metadata) (results.OperationResult[*roundtypes.NormalizedScorecard, error], error) {
	result, err := withTelemetry[*roundtypes.NormalizedScorecard, error](s, ctx, "NormalizeParsedScorecard", meta.RoundID, func(ctx context.Context) (results.OperationResult[*roundtypes.NormalizedScorecard, error], error) {
		s.logger.InfoContext(ctx, "Normalizing parsed scorecard",
			attr.String("import_id", meta.ImportID),
			attr.String("round_id", meta.RoundID.String()),
		)

		if data == nil {
			return results.FailureResult[*roundtypes.NormalizedScorecard, error](fmt.Errorf("parsed scorecard data is nil")), nil
		}

		// Infer mode if not set by parser
		mode := data.Mode
		if mode == "" {
			mode = sharedtypes.RoundModeSingles
			for _, p := range data.PlayerScores {
				if len(p.TeamNames) > 1 || p.IsTeam {
					mode = sharedtypes.RoundModeDoubles
					break
				}
			}
		}

		// 1. Initialize the Normalized structure defined in your roundtypes
		normalizedRound := roundtypes.NormalizedScorecard{
			ID:        uuid.NewString(), // Internal ID for this normalization instance
			RoundID:   meta.RoundID,
			GuildID:   meta.GuildID,
			ImportID:  meta.ImportID,
			Mode:      mode,
			ParScores: cloneInts(data.ParScores),
			CreatedAt: time.Now().UTC(),
		}

		// 2. Map PlayerScoreRow to NormalizedPlayer or NormalizedTeam
		if mode == sharedtypes.RoundModeDoubles {
			for _, p := range data.PlayerScores {

				teamID := uuid.New()

				team := roundtypes.NormalizedTeam{
					TeamID:     teamID,
					Total:      p.Total,
					HoleScores: cloneInts(p.HoleScores),
				}

				for _, name := range p.TeamNames {
					trimmedName := strings.TrimSpace(name)
					team.Members = append(team.Members, roundtypes.TeamMember{
						RawName: trimmedName,
					})
					s.logger.InfoContext(ctx, "Added team member from TeamNames",
						attr.String("raw_name", trimmedName),
						attr.String("team_id", teamID.String()))
				}

				if len(team.Members) == 0 && p.PlayerName != "" {
					trimmedName := strings.TrimSpace(p.PlayerName)
					team.Members = append(team.Members, roundtypes.TeamMember{
						RawName: trimmedName,
					})
					s.logger.InfoContext(ctx, "Added team member from PlayerName fallback",
						attr.String("raw_name", trimmedName),
						attr.String("team_id", teamID.String()))
				}

				s.logger.InfoContext(ctx, "Created team",
					attr.String("team_id", teamID.String()),
					attr.Int("member_count", len(team.Members)),
					attr.Int("total_score", team.Total))

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

		s.logger.InfoContext(ctx, "Normalization complete",
			attr.String("mode", string(mode)),
			attr.Int("player_count", len(data.PlayerScores)),
		)

		return results.SuccessResult[*roundtypes.NormalizedScorecard, error](&normalizedRound), nil
	})

	return result, err
}

// resolveUserID attempts to find a user ID by normalized name using the UserLookup adapter.
func (s *RoundService) resolveUserID(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, normalizedName string) sharedtypes.DiscordID {
	// 1. Try finding by UDisc Display Name
	identity, err := s.userLookup.FindByNormalizedUDiscDisplayName(ctx, db, guildID, normalizedName)
	if err == nil && identity != nil {
		return identity.UserID
	}

	// 2. Try finding by UDisc Username
	identity, err = s.userLookup.FindByNormalizedUDiscUsername(ctx, db, guildID, normalizedName)
	if err == nil && identity != nil {
		return identity.UserID
	}

	return ""
}

func cloneInts(in []int) []int {
	if in == nil {
		return nil
	}
	out := make([]int, len(in))
	copy(out, in)
	return out
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// IngestNormalizedScorecard performs user matching and prepares the scores for final application.
func (s *RoundService) IngestNormalizedScorecard(ctx context.Context, req roundtypes.ImportIngestScorecardInput) (results.OperationResult[*roundtypes.IngestScorecardResult, error], error) {
	result, err := withTelemetry[*roundtypes.IngestScorecardResult, error](s, ctx, "IngestNormalizedScorecard", req.RoundID, func(ctx context.Context) (results.OperationResult[*roundtypes.IngestScorecardResult, error], error) {
		return runInTx[*roundtypes.IngestScorecardResult, error](s, ctx, func(ctx context.Context, tx bun.IDB) (results.OperationResult[*roundtypes.IngestScorecardResult, error], error) {
			s.logger.InfoContext(ctx, "Ingesting normalized scorecard",
				attr.String("import_id", req.ImportID),
				attr.String("mode", string(req.NormalizedData.Mode)),
				attr.Int("teams_count", len(req.NormalizedData.Teams)),
				attr.Int("players_count", len(req.NormalizedData.Players)),
			)

			var finalScores []sharedtypes.ScoreInfo
			matchedCount := 0
			unmatchedPlayers := make([]string, 0)
			groupsToCreate := []roundtypes.Participant{}

			// --- Handle Mode: Doubles / Teams ---
			if req.NormalizedData.Mode != sharedtypes.RoundModeSingles {
				for _, team := range req.NormalizedData.Teams {
					teamMatched := false
					for _, member := range team.Members {
						normalizedName := normalizeName(member.RawName)
						s.logger.InfoContext(ctx, "Resolving team member",
							attr.String("raw_name", member.RawName),
							attr.String("normalized_name", normalizedName))
						discordID := s.resolveUserID(ctx, tx, req.GuildID, normalizedName)

						// Prepare participant for DB group creation
						groupsToCreate = append(groupsToCreate, roundtypes.Participant{
							UserID:  discordID,
							RawName: member.RawName,
						})

						if discordID != "" {
							finalScores = append(finalScores, sharedtypes.ScoreInfo{
								UserID: discordID,
								Score:  sharedtypes.Score(team.Total),
								TeamID: team.TeamID,
							})
							matchedCount++
							teamMatched = true
						} else {
							// Guest user - include with RawName but empty UserID
							finalScores = append(finalScores, sharedtypes.ScoreInfo{
								UserID:  "",
								Score:   sharedtypes.Score(team.Total),
								TeamID:  team.TeamID,
								RawName: member.RawName,
							})
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
					hasGroups, err := s.repo.RoundHasGroups(ctx, tx, req.RoundID)
					if err != nil {
						return results.FailureResult[*roundtypes.IngestScorecardResult, error](fmt.Errorf("failed checking existing round groups: %w", err)), nil
					}

					if !hasGroups {
						if err := s.repo.CreateRoundGroups(ctx, tx, req.RoundID, groupsToCreate); err != nil {
							return results.FailureResult[*roundtypes.IngestScorecardResult, error](fmt.Errorf("failed creating round groups: %w", err)), nil
						}
					}
				}

			} else {
				// --- Singles mode ---
				for _, p := range req.NormalizedData.Players {
					normalizedName := normalizeName(p.DisplayName)
					discordID := s.resolveUserID(ctx, tx, req.GuildID, normalizedName)
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
				return results.FailureResult[*roundtypes.IngestScorecardResult, error](fmt.Errorf("no valid player scores matched")), nil
			}

			// --- Return IngestScorecardResult ---
			return results.SuccessResult[*roundtypes.IngestScorecardResult, error](&roundtypes.IngestScorecardResult{
				ImportID:         req.ImportID,
				GuildID:          req.GuildID,
				RoundID:          req.RoundID,
				UserID:           req.UserID,
				ChannelID:        req.ChannelID,
				ScoresIngested:   len(finalScores),
				MatchedPlayers:   matchedCount,
				UnmatchedPlayers: len(unmatchedPlayers),
				SkippedPlayers:   unmatchedPlayers,
				Scores:           finalScores,
				RoundMode:        req.NormalizedData.Mode,
				EventMessageID:   req.EventMessageID,
				Timestamp:        time.Now().UTC(),
			}), nil
		})
	})

	return result, err
}
