package roundservice

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/parsers"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
)

// =============================================================================
// IMPORT JOB CREATION
// =============================================================================

func (s *RoundService) CreateImportJob(ctx context.Context, payload roundevents.ScorecardUploadedPayloadV1,
) (results.OperationResult, error) {

	return s.withTelemetry(ctx, "CreateImportJob", payload.RoundID, func(ctx context.Context) (results.OperationResult, error) {
		now := time.Now().UTC()

		round, err := s.repo.GetRound(ctx, payload.GuildID, payload.RoundID)
		if err != nil || round == nil {
			return results.OperationResult{
				Failure: importFailure(payload, errCodeRoundNotFound, "round not found"),
			}, nil
		}

		round.ImportID = payload.ImportID
		round.ImportStatus = string(rounddb.ImportStatusPending)
		round.ImportUserID = payload.UserID
		round.ImportChannelID = payload.ChannelID
		round.ImportedAt = &now
		round.FileName = payload.FileName
		round.FileData = payload.FileData
		round.UDiscURL = payload.UDiscURL
		round.ImportNotes = payload.Notes

		if payload.FileData != nil {
			round.ImportType = string(rounddb.ImportTypeCSV)
		} else {
			round.ImportType = string(rounddb.ImportTypeURL)
		}

		if _, err := s.repo.UpdateRound(ctx, payload.GuildID, payload.RoundID, round); err != nil {
			return results.OperationResult{
				Failure: importFailure(payload, errCodeDBError, err.Error()),
			}, nil
		}

		return results.OperationResult{
			Success: &payload,
		}, nil
	})
}

// =============================================================================
// PARSING
// =============================================================================

func (s *RoundService) ParseScorecard(ctx context.Context, payload roundevents.ScorecardUploadedPayloadV1, fileData []byte) (results.OperationResult, error) {

	return s.withTelemetry(ctx, "ParseScorecard", payload.RoundID, func(ctx context.Context) (results.OperationResult, error) {

		_ = s.repo.UpdateImportStatus(ctx, payload.GuildID, payload.RoundID, payload.ImportID, "parsing", "", "")

		if len(fileData) == 0 && payload.FileURL != "" {
			data, err := s.downloadFile(ctx, payload.FileURL)
			if err != nil {
				return results.OperationResult{
					Failure: importFailure(payload, errCodeDownloadError, err.Error()),
				}, nil
			}
			fileData = data
		}

		parser, err := parsers.NewFactory().GetParser(payload.FileName)
		if err != nil {
			return results.OperationResult{
				Failure: importFailure(payload, errCodeUnsupported, err.Error()),
			}, nil
		}

		parsed, err := parser.Parse(fileData)
		if err != nil {
			return results.OperationResult{
				Failure: &roundevents.ScorecardParseFailedPayloadV1{
					GuildID:   payload.GuildID,
					RoundID:   payload.RoundID,
					ImportID:  payload.ImportID,
					UserID:    payload.UserID,
					ChannelID: payload.ChannelID,
					Error:     err.Error(),
					Timestamp: time.Now().UTC(),
				},
			}, nil
		}

		parsed.ImportID = payload.ImportID
		parsed.GuildID = payload.GuildID
		parsed.RoundID = payload.RoundID

		_ = s.repo.UpdateImportStatus(ctx, payload.GuildID, payload.RoundID, payload.ImportID, "parsed", "", "")

		return results.OperationResult{
			Success: &roundevents.ParsedScorecardPayloadV1{
				ImportID:       payload.ImportID,
				GuildID:        payload.GuildID,
				RoundID:        payload.RoundID,
				UserID:         payload.UserID,
				ChannelID:      payload.ChannelID,
				ParsedData:     parsed,
				EventMessageID: payload.MessageID,
				Timestamp:      time.Now().UTC(),
			},
		}, nil
	})
}

// ScorecardURLRequested handles the specific case of a user providing a UDisc URL.
// It normalizes the URL, updates the round record, and produces the payload to start the parse.
func (s *RoundService) ScorecardURLRequested(ctx context.Context, payload roundevents.ScorecardURLRequestedPayloadV1) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "ScorecardURLRequested", payload.RoundID, func(ctx context.Context) (results.OperationResult, error) {
		now := time.Now().UTC()

		s.logger.InfoContext(ctx, "Handling scorecard URL request",
			attr.String("import_id", payload.ImportID),
			attr.String("guild_id", string(payload.GuildID)),
			attr.String("udisc_url", payload.UDiscURL),
		)

		// 1. Fetch Round
		round, err := s.repo.GetRound(ctx, payload.GuildID, payload.RoundID)
		if err != nil || round == nil {
			msg := "round not found"
			if err != nil {
				msg = err.Error()
				s.logger.ErrorContext(ctx, "Failed to fetch round for URL import", attr.Error(err))
			}
			return results.OperationResult{
				// Map the URL payload to the Uploaded payload format for the failure helper
				Failure: importFailure(roundevents.ScorecardUploadedPayloadV1{
					GuildID: payload.GuildID, RoundID: payload.RoundID, ImportID: payload.ImportID, UserID: payload.UserID, ChannelID: payload.ChannelID,
				}, errCodeRoundNotFound, msg),
			}, nil
		}

		// 2. Normalize the UDisc URL
		normalizedURL, err := normalizeUDiscExportURL(payload.UDiscURL)
		if err != nil {
			if !strings.Contains(strings.ToLower(payload.UDiscURL), "udisc.com") {
				s.logger.WarnContext(ctx, "Invalid UDisc URL provided", attr.String("url", payload.UDiscURL))
				return results.OperationResult{
					Failure: importFailure(roundevents.ScorecardUploadedPayloadV1{
						GuildID: payload.GuildID, RoundID: payload.RoundID, ImportID: payload.ImportID, UserID: payload.UserID, ChannelID: payload.ChannelID,
					}, errCodeInvalidUDiscURL, err.Error()),
				}, nil
			}
			normalizedURL = payload.UDiscURL
		}

		// 3. Update DB State
		round.ImportID = payload.ImportID
		round.ImportStatus = string(rounddb.ImportStatusPending)
		round.ImportType = string(rounddb.ImportTypeURL)
		round.UDiscURL = normalizedURL
		round.ImportUserID = payload.UserID
		round.ImportedAt = &now
		round.ImportNotes = payload.Notes

		if _, err := s.repo.UpdateRound(ctx, payload.GuildID, payload.RoundID, round); err != nil {
			s.logger.ErrorContext(ctx, "Failed to update round with normalized URL",
				attr.String("import_id", payload.ImportID),
				attr.Error(err))
			return results.OperationResult{
				Failure: importFailure(roundevents.ScorecardUploadedPayloadV1{
					GuildID: payload.GuildID, RoundID: payload.RoundID, ImportID: payload.ImportID, UserID: payload.UserID, ChannelID: payload.ChannelID,
				}, errCodeDBError, err.Error()),
			}, nil
		}

		s.logger.InfoContext(ctx, "UDisc URL request processed successfully",
			attr.String("import_id", payload.ImportID),
			attr.String("normalized_url", normalizedURL))

		// 4. Return success payload for the Parser
		return results.OperationResult{
			Success: &roundevents.ScorecardUploadedPayloadV1{
				GuildID:   payload.GuildID,
				RoundID:   payload.RoundID,
				ImportID:  payload.ImportID,
				UserID:    payload.UserID,
				ChannelID: payload.ChannelID,
				MessageID: payload.MessageID,
				FileURL:   normalizedURL,
				UDiscURL:  payload.UDiscURL,
				FileName:  "udisc_export.xlsx",
				Notes:     payload.Notes,
				Timestamp: now,
			},
		}, nil
	})
}

// =============================================================================
// NORMALIZATION
// =============================================================================

// NormalizeParsedScorecard converts a ParsedScorecard into a deterministic,
// ingestion-ready structure.
//
// Responsibilities:
//   - Detect doubles vs singles format
//   - Preserve hole ordering
//   - Preserve per-player/team totals
//   - For doubles: preserve team structure
//   - For singles: expand team rows into individual logical players
//   - Perform NO user matching
// NOTE: Normalization/ingestion/application functions were moved to dedicated files:
// - import_processing.go
// - import_application.go

// =============================================================================
// HELPERS
// =============================================================================

func (s *RoundService) downloadFile(ctx context.Context, url string) ([]byte, error) {
	req, err := newDownloadRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	resp, err := newDownloadClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, maxFileSize+1)
	buf, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}

	if len(buf) > maxFileSize {
		return nil, fmt.Errorf("file too large")
	}

	return buf, nil
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// resolveUserID attempts to match a normalized UDisc name to a Discord user ID.
// Returns empty string if no match found.
// resolveUserID is implemented in import_processing.go

func importFailure(p roundevents.ScorecardUploadedPayloadV1, code, msg string) *roundevents.ImportFailedPayloadV1 {
	return &roundevents.ImportFailedPayloadV1{
		GuildID:   p.GuildID,
		RoundID:   p.RoundID,
		ImportID:  p.ImportID,
		UserID:    p.UserID,
		ChannelID: p.ChannelID,
		Error:     msg,
		ErrorCode: code,
		Timestamp: time.Now().UTC(),
	}
}
func importFailureFromParsed(p roundevents.ParsedScorecardPayloadV1, code, msg string) *roundevents.ImportFailedPayloadV1 {
	return &roundevents.ImportFailedPayloadV1{
		GuildID:   p.GuildID,
		RoundID:   p.RoundID,
		ImportID:  p.ImportID,
		UserID:    p.UserID,
		ChannelID: p.ChannelID,
		Error:     msg,
		ErrorCode: code,
		Timestamp: time.Now().UTC(),
	}
}

func importFailureFromNormalized(p roundevents.ScorecardNormalizedPayloadV1, code, msg string) *roundevents.ImportFailedPayloadV1 {
	return &roundevents.ImportFailedPayloadV1{
		GuildID:   p.GuildID,
		RoundID:   p.RoundID,
		ImportID:  p.ImportID,
		UserID:    p.UserID,
		ChannelID: p.ChannelID,
		Error:     msg,
		ErrorCode: code,
		Timestamp: time.Now().UTC(),
	}
}

// Apply/import application functions moved to import_application.go

func cloneInts(in []int) []int {
	if len(in) == 0 {
		return nil
	}
	out := make([]int, len(in))
	copy(out, in)
	return out
}
