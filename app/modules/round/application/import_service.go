package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/parsers"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
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
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
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
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
					Error:     "round not found",
					ErrorCode: "ROUND_NOT_FOUND",
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		// Idempotency and conflict checks
		if round.ImportID != "" && round.ImportID != payload.ImportID {
			canOverwrite := false

			// 1) Completed/failed imports can be retried/overwritten.
			switch round.ImportStatus {
			case string(rounddb.ImportStatusCompleted), string(rounddb.ImportStatusFailed):
				canOverwrite = true
			}

			// 2) Same-user retries are allowed even if the previous attempt is still "in progress".
			// This helps recover from silent downstream failures where import status never reaches "completed".
			if !canOverwrite && round.ImportUserID != "" && payload.UserID != "" && round.ImportUserID == payload.UserID {
				canOverwrite = true
			}

			// 3) Stale imports can be overwritten regardless of status.
			// Keep this threshold relatively short so users can retry quickly.
			const staleThreshold = 2 * time.Minute
			if !canOverwrite && round.ImportedAt != nil && time.Since(*round.ImportedAt) > staleThreshold {
				canOverwrite = true
				s.logger.InfoContext(ctx, "Overwriting stale import",
					attr.String("old_import_id", round.ImportID),
					attr.String("old_status", round.ImportStatus),
					attr.Time("imported_at", *round.ImportedAt),
				)
			}

			// 4) Legacy safety valve: if the existing import record doesn't have enough metadata
			// to apply the checks above (e.g., older deployments didn't persist ImportUserID/ImportedAt),
			// allow overwrite so users aren't permanently blocked.
			if !canOverwrite && (round.ImportUserID == "" || round.ImportedAt == nil) {
				canOverwrite = true
				s.logger.InfoContext(ctx, "Overwriting import with missing metadata",
					attr.String("old_import_id", round.ImportID),
					attr.String("old_status", round.ImportStatus),
					attr.String("old_import_user_id", string(round.ImportUserID)),
					attr.Bool("old_has_imported_at", round.ImportedAt != nil),
				)
			}

			if !canOverwrite {
				importedAtStr := ""
				importAgeSeconds := int64(0)
				if round.ImportedAt != nil {
					importedAtStr = round.ImportedAt.Format(time.RFC3339)
					importAgeSeconds = int64(time.Since(*round.ImportedAt).Seconds())
				}

				s.logger.WarnContext(ctx, "Import ID conflict",
					attr.String("existing_import_id", round.ImportID),
					attr.String("incoming_import_id", payload.ImportID),
					attr.String("existing_status", round.ImportStatus),
					attr.String("existing_import_user_id", string(round.ImportUserID)),
					attr.String("incoming_user_id", string(payload.UserID)),
					attr.String("existing_imported_at", importedAtStr),
					attr.Int64("existing_import_age_seconds", importAgeSeconds),
				)
				return RoundOperationResult{
					Failure: &roundevents.ImportFailedPayload{
						GuildID:   payload.GuildID,
						RoundID:   payload.RoundID,
						ImportID:  payload.ImportID,
						UserID:    payload.UserID,
						ChannelID: payload.ChannelID,
						Error:     "another import is already in progress or completed",
						ErrorCode: "IMPORT_CONFLICT",
						Timestamp: time.Now().UTC(),
					},
				}, nil
			}
		}

		now := time.Now().UTC()

		if round.ImportID == payload.ImportID && string(round.ImportStatus) == string(rounddb.ImportStatusCompleted) {
			s.logger.InfoContext(ctx, "Import already completed; acknowledging idempotently", attr.String("import_id", payload.ImportID))
			return RoundOperationResult{
				Success: &roundevents.ScorecardUploadedPayload{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
					MessageID: payload.MessageID,
					FileData:  payload.FileData,
					FileURL:   payload.FileURL,
					FileName:  payload.FileName,
					UDiscURL:  payload.UDiscURL,
					Notes:     payload.Notes,
					Timestamp: now,
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
			// Note: Round domain model does not currently persist FileURL separately.
			// Keep FileData empty here to avoid corrupting parsing; the parse step will use payload.FileURL.
			round.ImportType = string(rounddb.ImportTypeCSV)
		}
		if payload.UDiscURL != "" {
			round.UDiscURL = payload.UDiscURL
			round.ImportType = string(rounddb.ImportTypeURL)
		}
		round.FileName = payload.FileName
		round.ImportNotes = payload.Notes
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
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
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
				GuildID:   payload.GuildID,
				RoundID:   payload.RoundID,
				ImportID:  payload.ImportID,
				UserID:    payload.UserID,
				ChannelID: payload.ChannelID,
				MessageID: payload.MessageID,
				FileData:  payload.FileData,
				FileURL:   payload.FileURL,
				FileName:  payload.FileName,
				UDiscURL:  payload.UDiscURL,
				Notes:     payload.Notes,
				Timestamp: now,
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
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
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
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
					Error:     "round not found",
					ErrorCode: "ROUND_NOT_FOUND",
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		now := time.Now().UTC()

		// Update the round with import context + UDisc URL information
		round.ImportID = payload.ImportID
		round.ImportStatus = string(rounddb.ImportStatusPending)
		round.ImportType = string(rounddb.ImportTypeURL)
		round.ImportUserID = payload.UserID
		round.ImportChannelID = payload.ChannelID

		// Ensure URL has /export appended if it's a UDisc scorecard URL
		uDiscURL := payload.UDiscURL
		if strings.Contains(uDiscURL, "udisc.com/scorecards/") && !strings.HasSuffix(uDiscURL, "/export") {
			uDiscURL = strings.TrimSuffix(uDiscURL, "/") + "/export"
		}
		round.UDiscURL = uDiscURL

		round.ImportNotes = payload.Notes
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
			// Success here is intended to be used as a parse request payload.
			// The parse handler expects ScorecardUploadedPayload.
			Success: &roundevents.ScorecardUploadedPayload{
				GuildID:   payload.GuildID,
				RoundID:   payload.RoundID,
				ImportID:  payload.ImportID,
				UserID:    payload.UserID,
				ChannelID: payload.ChannelID,
				MessageID: payload.MessageID,
				FileURL:   uDiscURL,
				FileName:  "udisc-export.csv",
				UDiscURL:  payload.UDiscURL,
				Notes:     payload.Notes,
				Timestamp: now,
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
			s.logger.InfoContext(ctx, "Downloading file from URL", attr.String("url", payload.FileURL))
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
			s.logger.InfoContext(ctx, "File downloaded successfully", attr.Int("size", len(fileData)))
		} else if len(fileData) == 0 {
			s.logger.WarnContext(ctx, "No file data provided and no URL available")
		} else {
			s.logger.InfoContext(ctx, "Using provided file data", attr.Int("size", len(fileData)))
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
				Failure: &roundevents.ScorecardParseFailedPayload{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
					Error:     fmt.Sprintf("failed to parse scorecard: %v", err),
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
		var matchedPlayersList []roundtypes.MatchedPlayer
		playersAutoAdded := 0
		autoAddedUserIDs := make([]sharedtypes.DiscordID, 0)

		parScores := payload.ParsedData.ParScores

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

			// The score module expects scores as relative-to-par (not raw stroke totals).
			// Compute total strokes and par for the holes the player actually played.
			totalStrokes := player.Total
			parForPlayer := 0
			if len(player.HoleScores) > 0 {
				sumStrokes := 0
				for i, strokes := range player.HoleScores {
					if strokes <= 0 {
						// Treat 0/negative as "not played / missing".
						continue
					}
					sumStrokes += strokes
					if i >= 0 && i < len(parScores) {
						parForPlayer += parScores[i]
					}
				}
				if totalStrokes == 0 {
					totalStrokes = sumStrokes
				}
			} else if totalStrokes > 0 {
				// No per-hole data; fall back to total and assume full par.
				for _, p := range parScores {
					parForPlayer += p
				}
			}

			scoreToPar := totalStrokes - parForPlayer

			var tagNumber *sharedtypes.TagNumber
			if tag, ok := tagByUser[userID]; ok {
				copyTag := *tag
				tagNumber = &copyTag
			} else {
				// Auto-add participant if not already in the round
				s.logger.InfoContext(ctx, "Auto-adding participant from scorecard",
					attr.String("import_id", payload.ImportID),
					attr.String("user_id", string(userID)),
				)

				newParticipant := roundtypes.Participant{
					UserID:   userID,
					Response: roundtypes.ResponseAccept, // Assume they accepted if they played
				}

				_, err := s.RoundDB.UpdateParticipant(ctx, payload.GuildID, payload.RoundID, newParticipant)
				if err != nil {
					s.logger.ErrorContext(ctx, "Failed to auto-add participant",
						attr.String("import_id", payload.ImportID),
						attr.String("user_id", string(userID)),
						attr.Error(err),
					)
					// Continue anyway, maybe they can be added later or it's a transient error
				} else {
					playersAutoAdded++
					autoAddedPayload := &roundevents.RoundParticipantAutoAddedPayload{
						RoundID:   payload.RoundID,
						GuildID:   payload.GuildID,
						UserID:    payload.UserID, // The user who initiated the import
						ChannelID: payload.ChannelID,
						AddedUser: userID, // The user being added
						ImportID:  payload.ImportID,
						Timestamp: time.Now().UTC(),
					}
					autoAddedUserIDs = append(autoAddedUserIDs, userID)

					payloadBytes, err := json.Marshal(autoAddedPayload)
					if err != nil {
						s.logger.ErrorContext(ctx, "Failed to marshal ParticipantAutoAdded payload",
							attr.Error(err),
						)
					} else {
						msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
						if err := s.EventBus.Publish(roundevents.RoundParticipantAutoAddedTopic, msg); err != nil {
							s.logger.ErrorContext(ctx, "Failed to publish ParticipantAutoAdded event",
								attr.Error(err),
							)
						}
					}
				}
			}

			scores = append(scores, sharedtypes.ScoreInfo{
				UserID:    userID,
				Score:     sharedtypes.Score(scoreToPar),
				TagNumber: tagNumber,
			})

			matchedPlayersList = append(matchedPlayersList, roundtypes.MatchedPlayer{
				DiscordID: userID,
				UDiscName: player.PlayerName,
				Score:     scoreToPar,
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
			attr.Int("auto_added_count", playersAutoAdded),
		)

		// Update status to indicate we're about to process scores
		_ = s.RoundDB.UpdateImportStatus(ctx, payload.GuildID, payload.RoundID, payload.ImportID, "processing", "", "")

		return RoundOperationResult{
			Success: &roundevents.ImportCompletedPayload{
				ImportID:           payload.ImportID,
				GuildID:            payload.GuildID,
				RoundID:            payload.RoundID,
				UserID:             payload.UserID,
				ChannelID:          payload.ChannelID,
				ScoresIngested:     len(scores),
				MatchedPlayers:     len(scores),
				UnmatchedPlayers:   len(unmatchedPlayers),
				PlayersAutoAdded:   playersAutoAdded,
				MatchedPlayersList: matchedPlayersList,
				SkippedPlayers:     unmatchedPlayers,
				AutoAddedUserIDs:   autoAddedUserIDs,
				Scores:             scores,
				Timestamp:          time.Now().UTC(),
			},
		}, nil
	})
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
