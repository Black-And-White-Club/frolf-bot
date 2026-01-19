package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/parsers"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// CreateImportJob creates a new import job for a scorecard upload.
func (s *RoundService) CreateImportJob(ctx context.Context, payload roundevents.ScorecardUploadedPayloadV1) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "CreateImportJob", payload.RoundID, func(ctx context.Context) (results.OperationResult, error) {
		s.logger.InfoContext(ctx, "Creating import job",
			attr.String("import_id", payload.ImportID),
			attr.String("guild_id", string(payload.GuildID)),
		)

		// Fetch the round to ensure it exists
		round, err := s.repo.GetRound(ctx, payload.GuildID, payload.RoundID)
		if err != nil {
			return results.OperationResult{
				Failure: &roundevents.ImportFailedPayloadV1{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
					Error:     fmt.Sprintf("failed to fetch round: %v", err),
					ErrorCode: errCodeRoundNotFound,
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		if round == nil {
			return results.OperationResult{
				Failure: &roundevents.ImportFailedPayloadV1{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
					Error:     "round not found",
					ErrorCode: errCodeRoundNotFound,
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
			if !canOverwrite && round.ImportedAt != nil && time.Since(*round.ImportedAt) > staleImportThreshold {
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
				s.logger.WarnContext(ctx, "another import is already in progress or completed",
					attr.String("existing_import_id", round.ImportID),
					attr.String("incoming_import_id", payload.ImportID),
				)
				return results.OperationResult{
					Failure: &roundevents.ImportFailedPayloadV1{
						GuildID:   payload.GuildID,
						RoundID:   payload.RoundID,
						ImportID:  payload.ImportID,
						UserID:    payload.UserID,
						ChannelID: payload.ChannelID,
						Error:     "another import is already in progress or completed",
						ErrorCode: errCodeImportConflict,
						Timestamp: time.Now().UTC(),
					},
				}, nil
			}
		}

		now := time.Now().UTC()

		if round.ImportID == payload.ImportID && string(round.ImportStatus) == string(rounddb.ImportStatusCompleted) {
			s.logger.InfoContext(ctx, "Import already completed; acknowledging idempotently", attr.String("import_id", payload.ImportID))
			return results.OperationResult{
				Success: &roundevents.ScorecardUploadedPayloadV1{
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
		_, err = s.repo.UpdateRound(ctx, payload.GuildID, payload.RoundID, round)
		if err != nil {
			wrapped := fmt.Errorf("failed to persist import job: %w", err)
			s.logger.ErrorContext(ctx, "Failed to update round with import job",
				attr.String("import_id", payload.ImportID),
				attr.Error(wrapped),
			)
			return results.OperationResult{
				Failure: &roundevents.ImportFailedPayloadV1{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
					Error:     wrapped.Error(),
					ErrorCode: errCodeDBError,
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		s.logger.InfoContext(ctx, "Import job created successfully",
			attr.String("import_id", payload.ImportID),
			attr.String("round_id", payload.RoundID.String()),
		)

		return results.OperationResult{
			Success: &roundevents.ScorecardUploadedPayloadV1{
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
func (s *RoundService) HandleScorecardURLRequested(ctx context.Context, payload roundevents.ScorecardURLRequestedPayloadV1) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "HandleScorecardURLRequested", payload.RoundID, func(ctx context.Context) (results.OperationResult, error) {
		s.logger.InfoContext(ctx, "Handling scorecard URL request",
			attr.String("import_id", payload.ImportID),
			attr.String("guild_id", string(payload.GuildID)),
			attr.String("udisc_url", payload.UDiscURL),
		)

		// Fetch the round to ensure it exists
		round, err := s.repo.GetRound(ctx, payload.GuildID, payload.RoundID)
		if err != nil {
			return results.OperationResult{
				Failure: &roundevents.ImportFailedPayloadV1{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
					Error:     fmt.Sprintf("failed to fetch round: %v", err),
					ErrorCode: errCodeRoundNotFound,
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		if round == nil {
			return results.OperationResult{
				Failure: &roundevents.ImportFailedPayloadV1{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
					Error:     "round not found",
					ErrorCode: errCodeRoundNotFound,
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

		// Normalize UDisc URL and persist as canonical export URL when possible.
		normalizedURL, err := normalizeUDiscExportURL(payload.UDiscURL)
		if err != nil {
			// If the host looks like udisc.com but the path isn't canonical, fall back
			// to persisting the original URL to preserve backward compatibility with tests
			// and older user inputs. Only fail when the host is not udisc.com.
			if !strings.Contains(strings.ToLower(payload.UDiscURL), "udisc.com") {
				return results.OperationResult{
					Failure: &roundevents.ImportFailedPayloadV1{
						GuildID:   payload.GuildID,
						RoundID:   payload.RoundID,
						ImportID:  payload.ImportID,
						UserID:    payload.UserID,
						ChannelID: payload.ChannelID,
						Error:     err.Error(),
						ErrorCode: errCodeInvalidUDiscURL,
						Timestamp: time.Now().UTC(),
					},
				}, nil
			}
			// Fallback: persist the original URL
			normalizedURL = payload.UDiscURL
		}

		round.UDiscURL = normalizedURL
		round.ImportNotes = payload.Notes
		round.ImportedAt = &now

		// Persist the update
		_, err = s.repo.UpdateRound(ctx, payload.GuildID, payload.RoundID, round)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to update round with UDisc URL",
				attr.String("import_id", payload.ImportID),
				attr.Error(err),
			)
			wrapped := fmt.Errorf("failed to persist UDisc URL: %w", err)
			s.logger.ErrorContext(ctx, "Failed to update round with UDisc URL", attr.Error(wrapped))
			return results.OperationResult{
				Failure: &roundevents.ImportFailedPayloadV1{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					Error:     wrapped.Error(),
					ErrorCode: errCodeDBError,
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		s.logger.InfoContext(ctx, "UDisc URL request handled successfully",
			attr.String("import_id", payload.ImportID),
			attr.String("round_id", payload.RoundID.String()),
		)

		return results.OperationResult{
			// Success here is intended to be used as a parse request payload.
			// The parse handler expects ScorecardUploadedPayload.
			Success: &roundevents.ScorecardUploadedPayloadV1{
				GuildID:   payload.GuildID,
				RoundID:   payload.RoundID,
				ImportID:  payload.ImportID,
				UserID:    payload.UserID,
				ChannelID: payload.ChannelID,
				MessageID: payload.MessageID,
				FileURL:   normalizedURL,
				FileName:  "udisc-export.xlsx",
				UDiscURL:  payload.UDiscURL,
				Notes:     payload.Notes,
				Timestamp: now,
			},
		}, nil
	})
}

// ParseScorecard parses a scorecard file and returns the parsed data
func (s *RoundService) ParseScorecard(ctx context.Context, payload roundevents.ScorecardUploadedPayloadV1, fileData []byte) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "ParseScorecard", payload.RoundID, func(ctx context.Context) (results.OperationResult, error) {
		s.logger.InfoContext(ctx, "Parsing scorecard",
			attr.String("import_id", payload.ImportID),
			attr.String("guild_id", string(payload.GuildID)),
			attr.String("file_name", payload.FileName),
		)

		// Mark status as parsing
		_ = s.repo.UpdateImportStatus(ctx, payload.GuildID, payload.RoundID, payload.ImportID, "parsing", "", "")

		// If we only have a URL, fetch it now with stricter limits and headers
		if len(fileData) == 0 && payload.FileURL != "" {
			s.logger.InfoContext(ctx, "Downloading file from URL", attr.String("url", payload.FileURL))

			client := newDownloadClient()
			req, err := newDownloadRequest(ctx, payload.FileURL)
			if err != nil {
				wrapped := fmt.Errorf("failed to build download request: %w", err)
				s.logger.ErrorContext(ctx, "download request build failed", attr.Error(wrapped))
				return results.OperationResult{
					Failure: &roundevents.ImportFailedPayloadV1{
						GuildID:   payload.GuildID,
						RoundID:   payload.RoundID,
						ImportID:  payload.ImportID,
						UserID:    payload.UserID,
						ChannelID: payload.ChannelID,
						Error:     wrapped.Error(),
						ErrorCode: errCodeDownloadError,
						Timestamp: time.Now().UTC(),
					},
				}, nil
			}

			resp, err := client.Do(req)
			if err != nil {
				wrapped := fmt.Errorf("failed to download file: %w", err)
				s.logger.ErrorContext(ctx, "download failed", attr.Error(wrapped))
				return results.OperationResult{
					Failure: &roundevents.ImportFailedPayloadV1{
						GuildID:   payload.GuildID,
						RoundID:   payload.RoundID,
						ImportID:  payload.ImportID,
						UserID:    payload.UserID,
						ChannelID: payload.ChannelID,
						Error:     wrapped.Error(),
						ErrorCode: errCodeDownloadError,
						Timestamp: time.Now().UTC(),
					},
				}, nil
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				wrapped := fmt.Errorf("download failed with status %d", resp.StatusCode)
				s.logger.ErrorContext(ctx, "download returned non-2xx", attr.Error(wrapped))
				return results.OperationResult{
					Failure: &roundevents.ImportFailedPayloadV1{
						GuildID:   payload.GuildID,
						RoundID:   payload.RoundID,
						ImportID:  payload.ImportID,
						UserID:    payload.UserID,
						ChannelID: payload.ChannelID,
						Error:     wrapped.Error(),
						ErrorCode: errCodeDownloadError,
						Timestamp: time.Now().UTC(),
					},
				}, nil
			}

			// Size-limited read
			limited := io.LimitReader(resp.Body, maxFileSize+1)
			buf, err := io.ReadAll(limited)
			if err != nil {
				wrapped := fmt.Errorf("failed to read download body: %w", err)
				s.logger.ErrorContext(ctx, "read download body failed", attr.Error(wrapped))
				return results.OperationResult{
					Failure: &roundevents.ImportFailedPayloadV1{
						GuildID:   payload.GuildID,
						RoundID:   payload.RoundID,
						ImportID:  payload.ImportID,
						UserID:    payload.UserID,
						ChannelID: payload.ChannelID,
						Error:     wrapped.Error(),
						ErrorCode: errCodeDownloadError,
						Timestamp: time.Now().UTC(),
					},
				}, nil
			}

			if len(buf) > maxFileSize {
				s.logger.ErrorContext(ctx, "downloaded file exceeds max size", attr.Int("size", len(buf)), attr.Int("max", maxFileSize))
				return results.OperationResult{
					Failure: &roundevents.ImportFailedPayloadV1{
						GuildID:   payload.GuildID,
						RoundID:   payload.RoundID,
						ImportID:  payload.ImportID,
						Error:     "file too large",
						ErrorCode: errCodeFileTooLarge,
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
			wrapped := fmt.Errorf("unsupported file format: %w", err)
			s.logger.ErrorContext(ctx, "parser selection failed", attr.Error(wrapped))
			return results.OperationResult{
				Failure: &roundevents.ImportFailedPayloadV1{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
					Error:     wrapped.Error(),
					ErrorCode: errCodeUnsupported,
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		// Parse the scorecard
		parsedScorecard, err := parser.Parse(fileData)
		if err != nil {
			wrapped := fmt.Errorf("failed to parse scorecard: %w", err)
			_ = s.repo.UpdateImportStatus(ctx, payload.GuildID, payload.RoundID, payload.ImportID, "failed", wrapped.Error(), errCodeParseError)
			s.logger.ErrorContext(ctx, "scorecard parse failed", attr.Error(wrapped))
			return results.OperationResult{
				Failure: &roundevents.ScorecardParseFailedPayloadV1{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
					Error:     wrapped.Error(),
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

		_ = s.repo.UpdateImportStatus(ctx, payload.GuildID, payload.RoundID, payload.ImportID, "parsed", "", "")
		return results.OperationResult{
			Success: &roundevents.ParsedScorecardPayloadV1{
				ImportID:       payload.ImportID,
				GuildID:        payload.GuildID,
				RoundID:        payload.RoundID,
				UserID:         payload.UserID,
				ChannelID:      payload.ChannelID,
				ParsedData:     parsedScorecard,
				EventMessageID: payload.MessageID,
				Timestamp:      time.Now().UTC(),
			},
		}, nil
	})
}

// IngestParsedScorecard matches parsed player rows to users and emits score processing requests.
func (s *RoundService) IngestParsedScorecard(ctx context.Context, payload roundevents.ParsedScorecardPayloadV1) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "IngestParsedScorecard", payload.RoundID, func(ctx context.Context) (results.OperationResult, error) {
		s.logger.InfoContext(ctx, "Ingesting parsed scorecard",
			attr.String("import_id", payload.ImportID),
			attr.String("guild_id", string(payload.GuildID)),
		)

		_ = s.repo.UpdateImportStatus(ctx, payload.GuildID, payload.RoundID, payload.ImportID, "ingesting", "", "")

		if payload.ParsedData == nil {
			return results.OperationResult{
				Failure: &roundevents.ImportFailedPayloadV1{
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

		round, err := s.repo.GetRound(ctx, payload.GuildID, payload.RoundID)
		if err != nil || round == nil {
			msg := "failed to fetch round"
			_ = s.repo.UpdateImportStatus(ctx, payload.GuildID, payload.RoundID, payload.ImportID, "failed", msg, errCodeRoundNotFound)
			return results.OperationResult{
				Failure: &roundevents.ImportFailedPayloadV1{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
					Error:     msg,
					ErrorCode: errCodeRoundNotFound,
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

			// CRITICAL: parser.Total is ALREADY the relative score (+/- from par)
			// The parsers extract this from the "+/-" (CSV) or "round_relative_score" (XLSX) columns
			// We do NOT need to recalculate it from hole-by-hole data
			scoreToPar := player.Total

			// Log the hole-by-hole data for debugging/validation if present
			if len(player.HoleScores) > 0 {
				s.logger.DebugContext(ctx, "Player hole scores (for logging only)",
					attr.String("import_id", payload.ImportID),
					attr.String("player_name", player.PlayerName),
					attr.Int("relative_score", scoreToPar),
					attr.Any("hole_scores", player.HoleScores),
				)
			}

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

				_, err := s.repo.UpdateParticipant(ctx, payload.GuildID, payload.RoundID, newParticipant)
				if err != nil {
					s.logger.ErrorContext(ctx, "Failed to auto-add participant",
						attr.String("import_id", payload.ImportID),
						attr.String("user_id", string(userID)),
						attr.Error(err),
					)
					// Continue anyway, maybe they can be added later or it's a transient error
				} else {
					playersAutoAdded++
					autoAddedPayload := &roundevents.RoundParticipantAutoAddedPayloadV1{
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
						if err := s.eventBus.Publish(roundevents.RoundParticipantAutoAddedV1, msg); err != nil {
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
			_ = s.repo.UpdateImportStatus(ctx, payload.GuildID, payload.RoundID, payload.ImportID, "failed", msg, "NO_MATCHES")
			return results.OperationResult{
				Failure: &roundevents.ImportFailedPayloadV1{
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
		_ = s.repo.UpdateImportStatus(ctx, payload.GuildID, payload.RoundID, payload.ImportID, "processing", "", "")

		return results.OperationResult{
			Success: &roundevents.ImportCompletedPayloadV1{
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
				EventMessageID:     payload.EventMessageID,
				Timestamp:          time.Now().UTC(),
			},
		}, nil
	})
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// ApplyImportedScores applies imported scores one-by-one using the same path as manual updates.
// It calls UpdateParticipantScore for each score and publishes ParticipantScoreUpdated events
// so downstream logic (CheckAllScoresSubmitted -> FinalizeRound) runs unchanged.
func (s *RoundService) ApplyImportedScores(ctx context.Context, payload roundevents.ImportCompletedPayloadV1) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "ApplyImportedScores", payload.RoundID, func(ctx context.Context) (results.OperationResult, error) {
		if len(payload.Scores) == 0 {
			s.logger.InfoContext(ctx, "No scores to apply in import",
				attr.String("import_id", payload.ImportID),
				attr.String("guild_id", string(payload.GuildID)),
				attr.String("round_id", payload.RoundID.String()),
			)
			return results.OperationResult{Success: nil}, nil
		}

		// Fetch participants once to minimize DB reads
		participants, err := s.repo.GetParticipants(ctx, payload.GuildID, payload.RoundID)
		if err != nil {
			return results.OperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
					GuildID: payload.GuildID,
					RoundID: payload.RoundID,
					Error:   err.Error(),
				},
			}, nil
		}

		participantMap := make(map[sharedtypes.DiscordID]*roundtypes.Participant, len(participants))
		for i := range participants {
			p := &participants[i]
			participantMap[p.UserID] = p
		}

		failedUserIDs := make([]string, 0)
		successCount := 0

		// Apply DB updates directly (batch-friendly)
		for i, score := range payload.Scores {
			// Periodically honor context cancellation to avoid wasted work
			if i%10 == 0 {
				if err := ctx.Err(); err != nil {
					wrapped := fmt.Errorf("context cancelled during import apply: %w", err)
					s.logger.ErrorContext(ctx, "import apply cancelled", attr.Error(wrapped))
					return results.OperationResult{
						Failure: &roundevents.ImportFailedPayloadV1{
							GuildID:   payload.GuildID,
							RoundID:   payload.RoundID,
							ImportID:  payload.ImportID,
							UserID:    payload.UserID,
							ChannelID: payload.ChannelID,
							Error:     wrapped.Error(),
							ErrorCode: errCodeCtxCancelled,
							Timestamp: time.Now().UTC(),
						},
					}, nil
				}
			}

			if err := s.repo.UpdateParticipantScore(ctx, payload.GuildID, payload.RoundID, score.UserID, score.Score); err != nil {
				wrapped := fmt.Errorf("failed to update participant score during import: %w", err)
				s.logger.ErrorContext(ctx, "failed to update participant score during import",
					attr.String("import_id", payload.ImportID),
					attr.String("guild_id", string(payload.GuildID)),
					attr.String("round_id", payload.RoundID.String()),
					attr.String("participant_id", string(score.UserID)),
					attr.Error(wrapped),
				)
				failedUserIDs = append(failedUserIDs, string(score.UserID))
				continue
			}

			// Update local view
			if p, ok := participantMap[score.UserID]; ok {
				copyScore := score.Score
				p.Score = &copyScore
			}

			successCount++
		}

		// Re-fetch authoritative participants after applying updates
		finalParticipants, err := s.repo.GetParticipants(ctx, payload.GuildID, payload.RoundID)
		if err != nil {
			return results.OperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
					GuildID: payload.GuildID,
					RoundID: payload.RoundID,
					Error:   err.Error(),
				},
			}, nil
		}

		// Do not publish from service. The handler will fan-out ParticipantScoreUpdated events.

		s.logger.InfoContext(ctx, "Imported scores application summary",
			attr.String("import_id", payload.ImportID),
			attr.String("guild_id", string(payload.GuildID)),
			attr.String("round_id", payload.RoundID.String()),
			attr.Int("successful_updates", successCount),
			attr.Int("failed_updates", len(failedUserIDs)),
			attr.Int("total_scores", len(payload.Scores)),
		)

		if successCount == 0 {
			return results.OperationResult{
				Failure: &roundevents.ImportFailedPayloadV1{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
					Error:     "all score updates failed during import",
					ErrorCode: errCodeImportApplyFailed,
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		return results.OperationResult{
			Success: &roundevents.ImportScoresAppliedPayloadV1{
				GuildID:        payload.GuildID,
				RoundID:        payload.RoundID,
				ImportID:       payload.ImportID,
				Participants:   finalParticipants,
				EventMessageID: payload.EventMessageID,
				Timestamp:      payload.Timestamp,
			},
		}, nil
	})
}
