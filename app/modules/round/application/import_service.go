package roundservice

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/parsers"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
)

// CreateImportJob creates a new import job for a scorecard upload.
func (s *RoundService) CreateImportJob(ctx context.Context, payload roundevents.ScorecardUploadedPayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "CreateImportJob", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Creating import job",
			attr.String("import_id", payload.ImportID),
			attr.String("guild_id", string(payload.GuildID)),
		)

		// Fetch the round to ensure it exists
		round, err := s.RoundDB.GetRound(ctx, payload.GuildID, payload.RoundID)
		if err != nil {
			return RoundOperationResult{
				Failure: &roundevents.ImportFailedPayload{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					Error:     fmt.Sprintf("failed to fetch round: %v", err),
					ErrorCode: "ROUND_NOT_FOUND",
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		if round == nil {
			return RoundOperationResult{
				Failure: &roundevents.ImportFailedPayload{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					Error:     "round not found",
					ErrorCode: "ROUND_NOT_FOUND",
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		// Idempotency and conflict checks
		if round.ImportID != "" && round.ImportID != payload.ImportID {
			s.logger.WarnContext(ctx, "Import ID conflict", attr.String("existing_import_id", round.ImportID), attr.String("incoming_import_id", payload.ImportID))
			return RoundOperationResult{
				Failure: &roundevents.ImportFailedPayload{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					Error:     "another import is already in progress or completed",
					ErrorCode: "IMPORT_CONFLICT",
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		if round.ImportID == payload.ImportID && string(round.ImportStatus) == string(rounddb.ImportStatusCompleted) {
			s.logger.InfoContext(ctx, "Import already completed; acknowledging idempotently", attr.String("import_id", payload.ImportID))
			return RoundOperationResult{
				Success: &roundevents.ScorecardUploadedPayload{
					GuildID:  payload.GuildID,
					RoundID:  payload.RoundID,
					ImportID: payload.ImportID,
					FileName: payload.FileName,
					FileURL:  payload.FileURL,
					UDiscURL: payload.UDiscURL,
					Notes:    payload.Notes,
				},
			}, nil
		}

		// Update the round with import job information
		round.ImportID = payload.ImportID
		round.ImportStatus = string(rounddb.ImportStatusPending)
		round.ImportType = string(rounddb.ImportTypeURL)
		round.ImportUserID = payload.UserID
		round.ImportChannelID = payload.ChannelID
		if payload.FileData != nil {
			round.FileData = payload.FileData
			round.ImportType = string(rounddb.ImportTypeCSV)
		}
		if payload.FileURL != "" {
			round.FileData = []byte(payload.FileURL)
			round.ImportType = string(rounddb.ImportTypeCSV)
		}
		if payload.UDiscURL != "" {
			round.UDiscURL = payload.UDiscURL
			round.ImportType = string(rounddb.ImportTypeURL)
		}
		round.FileName = payload.FileName
		round.ImportNotes = payload.Notes
		now := time.Now().UTC()
		round.ImportedAt = &now

		// Persist the import job
		_, err = s.RoundDB.UpdateRound(ctx, payload.GuildID, payload.RoundID, round)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to update round with import job",
				attr.String("import_id", payload.ImportID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: &roundevents.ImportFailedPayload{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					Error:     fmt.Sprintf("failed to persist import job: %v", err),
					ErrorCode: "DB_ERROR",
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		s.logger.InfoContext(ctx, "Import job created successfully",
			attr.String("import_id", payload.ImportID),
			attr.String("round_id", payload.RoundID.String()),
		)

		return RoundOperationResult{
			Success: &roundevents.ScorecardUploadedPayload{
				GuildID:  payload.GuildID,
				RoundID:  payload.RoundID,
				ImportID: payload.ImportID,
				FileName: payload.FileName,
				FileURL:  payload.FileURL,
				UDiscURL: payload.UDiscURL,
				Notes:    payload.Notes,
			},
		}, nil
	})
}

// HandleScorecardURLRequested handles scorecard URL requested events.
func (s *RoundService) HandleScorecardURLRequested(ctx context.Context, payload roundevents.ScorecardURLRequestedPayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "HandleScorecardURLRequested", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Handling scorecard URL request",
			attr.String("import_id", payload.ImportID),
			attr.String("guild_id", string(payload.GuildID)),
			attr.String("udisc_url", payload.UDiscURL),
		)

		// Fetch the round to ensure it exists
		round, err := s.RoundDB.GetRound(ctx, payload.GuildID, payload.RoundID)
		if err != nil {
			return RoundOperationResult{
				Failure: &roundevents.ImportFailedPayload{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					Error:     fmt.Sprintf("failed to fetch round: %v", err),
					ErrorCode: "ROUND_NOT_FOUND",
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		if round == nil {
			return RoundOperationResult{
				Failure: &roundevents.ImportFailedPayload{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					Error:     "round not found",
					ErrorCode: "ROUND_NOT_FOUND",
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		// Update the round with UDisc URL information
		round.ImportID = payload.ImportID
		round.ImportStatus = "pending"
		round.ImportType = "url"
		round.UDiscURL = payload.UDiscURL
		round.ImportNotes = payload.Notes
		now := time.Now().UTC()
		round.ImportedAt = &now

		// Persist the update
		_, err = s.RoundDB.UpdateRound(ctx, payload.GuildID, payload.RoundID, round)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to update round with UDisc URL",
				attr.String("import_id", payload.ImportID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: &roundevents.ImportFailedPayload{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					Error:     fmt.Sprintf("failed to persist UDisc URL: %v", err),
					ErrorCode: "DB_ERROR",
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		s.logger.InfoContext(ctx, "UDisc URL request handled successfully",
			attr.String("import_id", payload.ImportID),
			attr.String("round_id", payload.RoundID.String()),
		)

		return RoundOperationResult{
			Success: &roundevents.ScorecardURLRequestedPayload{
				GuildID:  payload.GuildID,
				RoundID:  payload.RoundID,
				ImportID: payload.ImportID,
				UDiscURL: payload.UDiscURL,
				Notes:    payload.Notes,
			},
		}, nil
	})
}

// ParseScorecard parses a scorecard file and returns the parsed data
func (s *RoundService) ParseScorecard(ctx context.Context, payload roundevents.ScorecardUploadedPayload, fileData []byte) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "ParseScorecard", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Parsing scorecard",
			attr.String("import_id", payload.ImportID),
			attr.String("guild_id", string(payload.GuildID)),
			attr.String("file_name", payload.FileName),
		)

		// Mark status as parsing
		_ = s.RoundDB.UpdateImportStatus(ctx, payload.GuildID, payload.RoundID, payload.ImportID, "parsing", "", "")

		// If we only have a URL, fetch it now
		if len(fileData) == 0 && payload.FileURL != "" {
			client := &http.Client{Timeout: 15 * time.Second}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, payload.FileURL, nil)
			if err != nil {
				return RoundOperationResult{
					Failure: &roundevents.ImportFailedPayload{
						GuildID:   payload.GuildID,
						RoundID:   payload.RoundID,
						ImportID:  payload.ImportID,
						UserID:    payload.UserID,
						ChannelID: payload.ChannelID,
						Error:     fmt.Sprintf("failed to build download request: %v", err),
						ErrorCode: "DOWNLOAD_ERROR",
						Timestamp: time.Now().UTC(),
					},
				}, nil
			}

			resp, err := client.Do(req)
			if err != nil {
				return RoundOperationResult{
					Failure: &roundevents.ImportFailedPayload{
						GuildID:   payload.GuildID,
						RoundID:   payload.RoundID,
						ImportID:  payload.ImportID,
						UserID:    payload.UserID,
						ChannelID: payload.ChannelID,
						Error:     fmt.Sprintf("failed to download file: %v", err),
						ErrorCode: "DOWNLOAD_ERROR",
						Timestamp: time.Now().UTC(),
					},
				}, nil
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return RoundOperationResult{
					Failure: &roundevents.ImportFailedPayload{
						GuildID:   payload.GuildID,
						RoundID:   payload.RoundID,
						ImportID:  payload.ImportID,
						UserID:    payload.UserID,
						ChannelID: payload.ChannelID,
						Error:     fmt.Sprintf("download failed with status %d", resp.StatusCode),
						ErrorCode: "DOWNLOAD_ERROR",
						Timestamp: time.Now().UTC(),
					},
				}, nil
			}

			buf, err := io.ReadAll(resp.Body)
			if err != nil {
				return RoundOperationResult{
					Failure: &roundevents.ImportFailedPayload{
						GuildID:   payload.GuildID,
						RoundID:   payload.RoundID,
						ImportID:  payload.ImportID,
						UserID:    payload.UserID,
						ChannelID: payload.ChannelID,
						Error:     fmt.Sprintf("failed to read download body: %v", err),
						ErrorCode: "DOWNLOAD_ERROR",
						Timestamp: time.Now().UTC(),
					},
				}, nil
			}

			fileData = buf
		}

		// Get the appropriate parser based on file extension
		parserFactory := parsers.NewFactory()
		parser, err := parserFactory.GetParser(payload.FileName)
		if err != nil {
			return RoundOperationResult{
				Failure: &roundevents.ImportFailedPayload{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
					Error:     fmt.Sprintf("unsupported file format: %v", err),
					ErrorCode: "UNSUPPORTED_FORMAT",
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		// Parse the scorecard
		parsedScorecard, err := parser.Parse(fileData)
		if err != nil {
			_ = s.RoundDB.UpdateImportStatus(ctx, payload.GuildID, payload.RoundID, payload.ImportID, "failed", fmt.Sprintf("failed to parse scorecard: %v", err), "PARSE_ERROR")
			return RoundOperationResult{
				Failure: &roundevents.ImportFailedPayload{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
					Error:     fmt.Sprintf("failed to parse scorecard: %v", err),
					ErrorCode: "PARSE_ERROR",
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		// Populate metadata
		parsedScorecard.ImportID = payload.ImportID
		parsedScorecard.RoundID = payload.RoundID
		parsedScorecard.GuildID = payload.GuildID

		s.logger.InfoContext(ctx, "Scorecard parsed successfully",
			attr.String("import_id", payload.ImportID),
			attr.Int("num_players", len(parsedScorecard.PlayerScores)),
			attr.Int("num_holes", len(parsedScorecard.ParScores)),
		)

		_ = s.RoundDB.UpdateImportStatus(ctx, payload.GuildID, payload.RoundID, payload.ImportID, "parsed", "", "")
		return RoundOperationResult{
			Success: &roundevents.ParsedScorecardPayload{
				ImportID:   payload.ImportID,
				GuildID:    payload.GuildID,
				RoundID:    payload.RoundID,
				UserID:     payload.UserID,
				ChannelID:  payload.ChannelID,
				ParsedData: parsedScorecard,
				Timestamp:  time.Now().UTC(),
			},
		}, nil
	})
}

// IngestParsedScorecard matches parsed player rows to users and emits score processing requests.
func (s *RoundService) IngestParsedScorecard(ctx context.Context, payload roundevents.ParsedScorecardPayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "IngestParsedScorecard", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Ingesting parsed scorecard",
			attr.String("import_id", payload.ImportID),
			attr.String("guild_id", string(payload.GuildID)),
		)

		_ = s.RoundDB.UpdateImportStatus(ctx, payload.GuildID, payload.RoundID, payload.ImportID, "ingesting", "", "")

		if payload.ParsedData == nil {
			return RoundOperationResult{
				Failure: &roundevents.ImportFailedPayload{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
					Error:     "parsed data missing",
					ErrorCode: "PARSED_DATA_MISSING",
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		round, err := s.RoundDB.GetRound(ctx, payload.GuildID, payload.RoundID)
		if err != nil || round == nil {
			msg := "failed to fetch round"
			_ = s.RoundDB.UpdateImportStatus(ctx, payload.GuildID, payload.RoundID, payload.ImportID, "failed", msg, "ROUND_NOT_FOUND")
			return RoundOperationResult{
				Failure: &roundevents.ImportFailedPayload{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
					Error:     msg,
					ErrorCode: "ROUND_NOT_FOUND",
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		tagByUser := map[sharedtypes.DiscordID]*sharedtypes.TagNumber{}
		for _, p := range round.Participants {
			if p.TagNumber != nil {
				copyTag := *p.TagNumber
				tagByUser[p.UserID] = &copyTag
			}
		}

		var scores []sharedtypes.ScoreInfo
		var unmatchedPlayers []string

		// Match players and collect scores. Unmatched players are skipped (not an error).
		for _, player := range payload.ParsedData.PlayerScores {
			normalized := normalizeName(player.PlayerName)
			if normalized == "" {
				unmatchedPlayers = append(unmatchedPlayers, player.PlayerName)
				continue
			}

			var userID sharedtypes.DiscordID
			if s.userLookup != nil {
				if u, lookupErr := s.userLookup.FindByNormalizedUDiscUsername(ctx, payload.GuildID, normalized); lookupErr == nil && u != nil {
					userID = u.UserID
				}
				if userID == "" {
					if u, lookupErr := s.userLookup.FindByNormalizedUDiscDisplayName(ctx, payload.GuildID, normalized); lookupErr == nil && u != nil {
						userID = u.UserID
					}
				}
			}

			if userID == "" {
				unmatchedPlayers = append(unmatchedPlayers, player.PlayerName)
				s.logger.InfoContext(ctx, "Player not matched to Discord user",
					attr.String("import_id", payload.ImportID),
					attr.String("player_name", player.PlayerName),
				)
				continue
			}

			total := player.Total
			if total == 0 {
				for _, hole := range player.HoleScores {
					total += hole
				}
			}

			var tagNumber *sharedtypes.TagNumber
			if tag, ok := tagByUser[userID]; ok {
				copyTag := *tag
				tagNumber = &copyTag
			}

			scores = append(scores, sharedtypes.ScoreInfo{
				UserID:    userID,
				Score:     sharedtypes.Score(total),
				TagNumber: tagNumber,
			})
		}

		// If no matched players, fail the import
		if len(scores) == 0 {
			msg := fmt.Sprintf("no players matched (%d total in scorecard)", len(payload.ParsedData.PlayerScores))
			_ = s.RoundDB.UpdateImportStatus(ctx, payload.GuildID, payload.RoundID, payload.ImportID, "failed", msg, "NO_MATCHES")
			return RoundOperationResult{
				Failure: &roundevents.ImportFailedPayload{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
					Error:     msg,
					ErrorCode: "NO_MATCHES",
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		s.logger.InfoContext(ctx, "Scorecard ingestion complete - ready for score processing",
			attr.String("import_id", payload.ImportID),
			attr.Int("matched_count", len(scores)),
			attr.Int("unmatched_count", len(unmatchedPlayers)),
		)

		// Update status to indicate we're about to process scores
		_ = s.RoundDB.UpdateImportStatus(ctx, payload.GuildID, payload.RoundID, payload.ImportID, "processing", "", "")

		// Publish the request to process scores
		return RoundOperationResult{
			Success: &scoreevents.ProcessRoundScoresRequestPayload{
				GuildID: payload.GuildID,
				RoundID: payload.RoundID,
				Scores:  scores,
			},
		}, nil
	})
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
