package roundservice

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/parsers"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	"github.com/uptrace/bun"
)

// =============================================================================
// IMPORT JOB CREATION
// =============================================================================

func (s *RoundService) CreateImportJob(ctx context.Context, req *roundtypes.ImportCreateJobInput) (CreateImportJobResult, error) {
	result, err := withTelemetry[roundtypes.CreateImportJobResult, error](s, ctx, "CreateImportJob", req.RoundID, func(ctx context.Context) (CreateImportJobResult, error) {
		return runInTx[roundtypes.CreateImportJobResult, error](s, ctx, func(ctx context.Context, tx bun.IDB) (CreateImportJobResult, error) {
			now := time.Now().UTC()

			round, err := s.repo.GetRound(ctx, tx, req.GuildID, req.RoundID)
			if err != nil || round == nil {
				msg := "round not found"
				if err != nil {
					msg = err.Error()
				}
				return results.FailureResult[roundtypes.CreateImportJobResult, error](fmt.Errorf(msg)), nil
			}

			round.ImportID = req.ImportID
			round.ImportStatus = string(rounddb.ImportStatusPending)
			round.ImportUserID = req.UserID
			round.ImportChannelID = req.ChannelID
			round.ImportedAt = &now
			round.FileName = req.FileName
			round.FileData = req.FileData
			round.UDiscURL = req.UDiscURL
			round.ImportNotes = req.Notes

			if req.FileData != nil {
				round.ImportType = string(rounddb.ImportTypeCSV)
			} else {
				round.ImportType = string(rounddb.ImportTypeURL)
			}

			if _, err := s.repo.UpdateRound(ctx, tx, req.GuildID, req.RoundID, round); err != nil {
				return results.FailureResult[roundtypes.CreateImportJobResult, error](err), nil
			}

			return results.SuccessResult[roundtypes.CreateImportJobResult, error](roundtypes.CreateImportJobResult{Job: req}), nil
		})
	})

	return result, err
}

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

// =============================================================================
// PARSING
// =============================================================================

func (s *RoundService) ParseScorecard(ctx context.Context, req *roundtypes.ImportParseScorecardInput) (ParseScorecardResult, error) {
	result, err := withTelemetry[roundtypes.ParsedScorecard, error](s, ctx, "ParseScorecard", req.RoundID, func(ctx context.Context) (ParseScorecardResult, error) {

		_ = s.repo.UpdateImportStatus(ctx, nil, req.GuildID, req.RoundID, req.ImportID, "parsing", "", "")

		fileData := req.FileData
		if len(fileData) == 0 && req.FileURL != "" {
			data, err := s.downloadFile(ctx, req.FileURL)
			if err != nil {
				return results.FailureResult[roundtypes.ParsedScorecard, error](fmt.Errorf("Download error: %w", err)), nil
			}
			fileData = data
		}

		parser, err := parsers.NewFactory().GetParser(req.FileName)
		if err != nil {
			return results.FailureResult[roundtypes.ParsedScorecard, error](fmt.Errorf("Unsupported file type: %w", err)), nil
		}

		parsed, err := parser.Parse(fileData)
		if err != nil {
			return results.FailureResult[roundtypes.ParsedScorecard, error](err), nil
		}

		parsed.ImportID = req.ImportID
		parsed.GuildID = req.GuildID
		parsed.RoundID = req.RoundID

		_ = s.repo.UpdateImportStatus(ctx, nil, req.GuildID, req.RoundID, req.ImportID, "parsed", "", "")

		return results.SuccessResult[roundtypes.ParsedScorecard, error](*parsed), nil
	})

	return result, err
}

// ScorecardURLRequested handles the specific case of a user providing a UDisc URL.
func (s *RoundService) ScorecardURLRequested(ctx context.Context, req *roundtypes.ImportCreateJobInput) (CreateImportJobResult, error) {
	result, err := withTelemetry[roundtypes.CreateImportJobResult, error](s, ctx, "ScorecardURLRequested", req.RoundID, func(ctx context.Context) (CreateImportJobResult, error) {
		return runInTx[roundtypes.CreateImportJobResult, error](s, ctx, func(ctx context.Context, tx bun.IDB) (CreateImportJobResult, error) {
			now := time.Now().UTC()

			s.logger.InfoContext(ctx, "Handling scorecard URL request",
				attr.String("import_id", req.ImportID),
				attr.String("guild_id", string(req.GuildID)),
				attr.String("udisc_url", req.UDiscURL),
			)

			// 1. Fetch Round
			round, err := s.repo.GetRound(ctx, tx, req.GuildID, req.RoundID)
			if err != nil || round == nil {
				msg := "round not found"
				if err != nil {
					msg = err.Error()
					s.logger.ErrorContext(ctx, "Failed to fetch round for URL import", attr.Error(err))
				}
				return results.FailureResult[roundtypes.CreateImportJobResult, error](fmt.Errorf(msg)), nil
			}

			// 2. Normalize the UDisc URL
			normalizedURL, err := normalizeUDiscExportURL(req.UDiscURL)
			if err != nil {
				if !strings.Contains(strings.ToLower(req.UDiscURL), "udisc.com") {
					s.logger.WarnContext(ctx, "Invalid UDisc URL provided", attr.String("url", req.UDiscURL))
					return results.FailureResult[roundtypes.CreateImportJobResult, error](err), nil
				}
				normalizedURL = req.UDiscURL
			}

			// 3. Update DB State
			round.ImportID = req.ImportID
			round.ImportStatus = string(rounddb.ImportStatusPending)
			round.ImportType = string(rounddb.ImportTypeURL)
			round.UDiscURL = normalizedURL
			round.ImportUserID = req.UserID
			round.ImportedAt = &now
			round.ImportNotes = req.Notes

			if _, err := s.repo.UpdateRound(ctx, tx, req.GuildID, req.RoundID, round); err != nil {
				s.logger.ErrorContext(ctx, "Failed to update round with normalized URL",
					attr.String("import_id", req.ImportID),
					attr.Error(err))
				return results.FailureResult[roundtypes.CreateImportJobResult, error](err), nil
			}

			s.logger.InfoContext(ctx, "UDisc URL request processed successfully",
				attr.String("import_id", req.ImportID),
				attr.String("normalized_url", normalizedURL))

			// 4. Return success result
			// Update the req with the normalized URL so it propagates correctly
			req.UDiscURL = normalizedURL
			return results.SuccessResult[roundtypes.CreateImportJobResult, error](roundtypes.CreateImportJobResult{Job: req}), nil
		})
	})

	return result, err
}
