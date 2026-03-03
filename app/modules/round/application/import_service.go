package roundservice

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	"github.com/uptrace/bun"
)

// =============================================================================
// IMPORT JOB CREATION
// =============================================================================

func (s *RoundService) CreateImportJob(ctx context.Context, req *roundtypes.ImportCreateJobInput) (CreateImportJobResult, error) {
	result, err := withTelemetry(s, ctx, "CreateImportJob", req.RoundID, func(ctx context.Context) (CreateImportJobResult, error) {
		source := normalizeImportSource(req.Source)
		importInputKind := inputKind(req.FileData, req.FileURL, req.UDiscURL)
		importFileExt := fileExt(req.FileName, req.FileURL, req.UDiscURL)
		roundState := "unknown"

		s.importerMetrics.RecordImportAttempt(ctx, source, importInputKind, importFileExt, roundState)

		s.logger.InfoContext(ctx, "Creating import job",
			attr.String("import_id", req.ImportID),
			attr.String("guild_id", string(req.GuildID)),
			attr.String("round_id", req.RoundID.String()),
			attr.String("source", source),
			attr.String("input_kind", importInputKind),
			attr.String("file_ext", importFileExt),
			attr.Bool("has_file_data", len(req.FileData) > 0),
			attr.Bool("has_udisc_url", req.UDiscURL != ""),
		)

		return runInTx(s, ctx, func(ctx context.Context, tx bun.IDB) (CreateImportJobResult, error) {
			now := time.Now().UTC()

			round, err := s.repo.GetRound(ctx, tx, req.GuildID, req.RoundID)
			if err != nil || round == nil {
				msg := "round not found"
				if err != nil {
					msg = err.Error()
				}
				failureErr := fmt.Errorf("%s", msg)
				s.recordImportFailure(ctx, source, importInputKind, importFileExt, roundState, failureErr)
				return results.FailureResult[roundtypes.CreateImportJobResult](failureErr), nil
			}
			roundState = roundStateValue(round)

			if !isAdminImportSource(source) && round.State != roundtypes.RoundStateInProgress {
				failureErr := fmt.Errorf("round must be %s before scorecard imports (current state: %s)", roundtypes.RoundStateInProgress, round.State)
				s.recordImportFailure(ctx, source, importInputKind, importFileExt, roundState, failureErr)
				_ = s.repo.UpdateImportStatus(ctx, tx, req.GuildID, req.RoundID, req.ImportID, string(rounddb.ImportStatusFailed), failureErr.Error(), errCodeRoundStateInvalid)
				return results.FailureResult[roundtypes.CreateImportJobResult](failureErr), nil
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

			switch importFileExt {
			case ".xlsx":
				round.ImportType = string(rounddb.ImportTypeXLSX)
			case ".csv":
				round.ImportType = string(rounddb.ImportTypeCSV)
			case "unknown":
				if req.UDiscURL != "" {
					round.ImportType = string(rounddb.ImportTypeURL)
				} else {
					round.ImportType = string(rounddb.ImportTypeCSV)
				}
			default:
				round.ImportType = string(rounddb.ImportTypeCSV)
			}
			if req.UDiscURL != "" && importInputKind == "url" {
				round.ImportType = string(rounddb.ImportTypeURL)
			}

			if _, err := s.repo.UpdateRound(ctx, tx, req.GuildID, req.RoundID, round); err != nil {
				s.recordImportFailure(ctx, source, importInputKind, importFileExt, roundState, err)
				return results.FailureResult[roundtypes.CreateImportJobResult](err), nil
			}

			s.logger.InfoContext(ctx, "Import job created",
				attr.String("import_id", req.ImportID),
				attr.String("round_id", req.RoundID.String()),
				attr.String("source", source),
				attr.String("round_state", roundState),
			)

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

	client := s.downloadClient
	if client == nil {
		client = newDownloadClient()
	}

	resp, err := client.Do(req)
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
	result, err := withTelemetry(s, ctx, "ParseScorecard", req.RoundID, func(ctx context.Context) (ParseScorecardResult, error) {
		source := normalizeImportSource(req.Source)
		importInputKind := inputKindForRequest(source, req.FileData, req.FileURL)
		importFileExt := fileExt(req.FileName, req.FileURL, "")
		roundState := "unknown"

		_ = s.repo.UpdateImportStatus(ctx, nil, req.GuildID, req.RoundID, req.ImportID, "parsing", "", "")

		fileData := req.FileData
		if len(fileData) == 0 && req.FileURL != "" {
			downloadStart := time.Now()
			data, err := s.downloadFile(ctx, req.FileURL)
			if err != nil {
				failureErr := fmt.Errorf("download error: %w", err)
				_ = s.repo.UpdateImportStatus(ctx, nil, req.GuildID, req.RoundID, req.ImportID, string(rounddb.ImportStatusFailed), failureErr.Error(), errCodeDownloadError)
				s.recordImportFailure(ctx, source, importInputKind, importFileExt, roundState, failureErr)
				return results.FailureResult[roundtypes.ParsedScorecard](failureErr), nil
			}
			s.recordImportPhaseDuration(ctx, importPhaseDownload, source, importInputKind, importFileExt, time.Since(downloadStart))
			fileData = data
		}
		if len(fileData) > maxFileSize {
			failureErr := fmt.Errorf("file too large")
			_ = s.repo.UpdateImportStatus(ctx, nil, req.GuildID, req.RoundID, req.ImportID, string(rounddb.ImportStatusFailed), failureErr.Error(), errCodeFileTooLarge)
			s.recordImportFailure(ctx, source, importInputKind, importFileExt, roundState, failureErr)
			return results.FailureResult[roundtypes.ParsedScorecard](failureErr), nil
		}

		parser, err := s.parserFactory.GetParser(req.FileName)
		if err != nil {
			failureErr := fmt.Errorf("unsupported file type: %w", err)
			_ = s.repo.UpdateImportStatus(ctx, nil, req.GuildID, req.RoundID, req.ImportID, string(rounddb.ImportStatusFailed), failureErr.Error(), errCodeUnsupported)
			s.recordImportFailure(ctx, source, importInputKind, importFileExt, roundState, failureErr)
			return results.FailureResult[roundtypes.ParsedScorecard](failureErr), nil
		}

		parseStart := time.Now()
		parsed, err := parser.Parse(fileData)
		if err != nil {
			failureErr := fmt.Errorf("parse error: %w", err)
			_ = s.repo.UpdateImportStatus(ctx, nil, req.GuildID, req.RoundID, req.ImportID, string(rounddb.ImportStatusFailed), failureErr.Error(), errCodeParseError)
			s.recordImportFailure(ctx, source, importInputKind, importFileExt, roundState, failureErr)
			return results.FailureResult[roundtypes.ParsedScorecard](failureErr), nil
		}
		s.recordImportPhaseDuration(ctx, importPhaseParse, source, importInputKind, importFileExt, time.Since(parseStart))

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
	result, err := withTelemetry(s, ctx, "ScorecardURLRequested", req.RoundID, func(ctx context.Context) (CreateImportJobResult, error) {
		source := normalizeImportSource(req.Source)
		importInputKind := inputKind(req.FileData, req.FileURL, req.UDiscURL)
		importFileExt := fileExt(req.FileName, req.FileURL, req.UDiscURL)
		roundState := "unknown"

		s.importerMetrics.RecordImportAttempt(ctx, source, importInputKind, importFileExt, roundState)

		return runInTx(s, ctx, func(ctx context.Context, tx bun.IDB) (CreateImportJobResult, error) {
			now := time.Now().UTC()

			s.logger.InfoContext(ctx, "Handling scorecard URL request",
				attr.String("import_id", req.ImportID),
				attr.String("guild_id", string(req.GuildID)),
				attr.String("round_id", req.RoundID.String()),
				attr.String("source", source),
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
				failureErr := fmt.Errorf("%s", msg)
				s.recordImportFailure(ctx, source, importInputKind, importFileExt, roundState, failureErr)
				return results.FailureResult[roundtypes.CreateImportJobResult](failureErr), nil
			}
			roundState = roundStateValue(round)
			if !isAdminImportSource(source) && round.State != roundtypes.RoundStateInProgress {
				failureErr := fmt.Errorf("round must be %s before scorecard imports (current state: %s)", roundtypes.RoundStateInProgress, round.State)
				s.recordImportFailure(ctx, source, importInputKind, importFileExt, roundState, failureErr)
				_ = s.repo.UpdateImportStatus(ctx, tx, req.GuildID, req.RoundID, req.ImportID, string(rounddb.ImportStatusFailed), failureErr.Error(), errCodeRoundStateInvalid)
				return results.FailureResult[roundtypes.CreateImportJobResult](failureErr), nil
			}

			// 2. Normalize the UDisc URL
			normalizedURL, err := normalizeUDiscExportURL(req.UDiscURL)
			if err != nil {
				s.logger.WarnContext(ctx, "Invalid UDisc URL provided", attr.String("url", req.UDiscURL))
				failureErr := fmt.Errorf("invalid UDisc URL: %w", err)
				s.recordImportFailure(ctx, source, importInputKind, importFileExt, roundState, failureErr)
				return results.FailureResult[roundtypes.CreateImportJobResult](failureErr), nil
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
				s.recordImportFailure(ctx, source, importInputKind, importFileExt, roundState, err)
				return results.FailureResult[roundtypes.CreateImportJobResult](err), nil
			}

			s.logger.InfoContext(ctx, "UDisc URL request processed successfully",
				attr.String("import_id", req.ImportID),
				attr.String("source", source),
				attr.String("normalized_url", normalizedURL))

			// 4. Return success result
			// Update the req with the normalized URL so it propagates correctly
			req.UDiscURL = normalizedURL
			return results.SuccessResult[roundtypes.CreateImportJobResult, error](roundtypes.CreateImportJobResult{Job: req}), nil
		})
	})

	return result, err
}
