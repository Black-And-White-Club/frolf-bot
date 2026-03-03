package roundservice

import (
	"context"
	"errors"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const (
	importPhaseDownload = "download"
	importPhaseParse    = "parse"
	importPhaseMatch    = "match"
	importPhaseApply    = "apply"
)

func normalizeImportSource(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return "unknown"
	}
	return source
}

func isAdminImportSource(source string) bool {
	return normalizeImportSource(source) == importSourceAdminPWA
}

func inputKind(fileData []byte, fileURL, uDiscURL string) string {
	if len(fileData) > 0 || strings.TrimSpace(fileURL) != "" {
		return "file"
	}
	if strings.TrimSpace(uDiscURL) != "" {
		return "url"
	}
	return "unknown"
}

func fileExt(fileName, fileURL, uDiscURL string) string {
	if ext := normalizeExt(filepath.Ext(strings.TrimSpace(fileName))); ext != "" {
		return ext
	}
	if ext := extFromURL(fileURL); ext != "" {
		return ext
	}
	if ext := extFromURL(uDiscURL); ext != "" {
		return ext
	}
	return "unknown"
}

func extFromURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	return normalizeExt(filepath.Ext(parsed.Path))
}

func normalizeExt(ext string) string {
	ext = strings.TrimSpace(strings.ToLower(ext))
	if ext == "" {
		return ""
	}
	return ext
}

func roundStateValue(round *roundtypes.Round) string {
	if round == nil {
		return "unknown"
	}
	if round.State == "" {
		return "unknown"
	}
	return string(round.State)
}

func inputKindFromRound(round *roundtypes.Round) string {
	if round == nil {
		return "unknown"
	}
	if strings.TrimSpace(round.UDiscURL) != "" {
		return "url"
	}
	if len(round.FileData) > 0 || strings.TrimSpace(round.FileName) != "" {
		return "file"
	}
	return "unknown"
}

func defaultInputKindForSource(source string) string {
	switch normalizeImportSource(source) {
	case importSourceAdminPWA, importSourceDiscordUpload:
		return "file"
	case importSourceDiscordURL:
		return "url"
	default:
		return "unknown"
	}
}

func inputKindForRequest(source string, fileData []byte, fileURL string) string {
	if len(fileData) > 0 {
		return "file"
	}
	if strings.TrimSpace(fileURL) != "" {
		if normalizeImportSource(source) == importSourceDiscordURL {
			return "url"
		}
		return "file"
	}
	return defaultInputKindForSource(source)
}

func (s *RoundService) resolveImportContext(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	roundID sharedtypes.RoundID,
	source string,
) (importInputKind, importFileExt, roundState string) {
	importInputKind = defaultInputKindForSource(source)
	importFileExt = "unknown"
	roundState = "unknown"

	if s.repo == nil {
		return importInputKind, importFileExt, roundState
	}

	if guildID == "" || roundID.UUID() == uuid.Nil {
		return importInputKind, importFileExt, roundState
	}

	round, err := s.repo.GetRound(ctx, db, guildID, roundID)
	if err != nil || round == nil {
		return importInputKind, importFileExt, roundState
	}

	return inputKindFromRound(round), fileExt(round.FileName, "", round.UDiscURL), roundStateValue(round)
}

func classifyImportFailure(err error) string {
	if err == nil {
		return "unknown"
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "round not found"), errors.Is(err, ErrRoundNotFound):
		return "round_not_found"
	case strings.Contains(msg, "in_progress"):
		return "round_not_in_progress"
	case strings.Contains(msg, "invalid udisc"):
		return "invalid_udisc_url"
	case strings.Contains(msg, "unsupported file type"):
		return "unsupported_format"
	case strings.Contains(msg, "file too large"):
		return "file_too_large"
	case strings.Contains(msg, "download"):
		return "download_error"
	case strings.Contains(msg, "parse"):
		return "parse_error"
	case strings.Contains(msg, "normalize"):
		return "normalize_error"
	case strings.Contains(msg, "ingest"):
		return "ingest_error"
	case strings.Contains(msg, "apply"):
		return "apply_error"
	case strings.Contains(msg, "failed to update"):
		return "db_error"
	default:
		return "unknown_error"
	}
}

func (s *RoundService) recordImportFailure(
	ctx context.Context,
	source, importInputKind, importFileExt, roundState string,
	err error,
) {
	s.importerMetrics.RecordImportFailure(
		ctx,
		normalizeImportSource(source),
		valueOrUnknown(importInputKind),
		valueOrUnknown(importFileExt),
		valueOrUnknown(roundState),
		classifyImportFailure(err),
	)
}

func (s *RoundService) recordImportPhaseDuration(
	ctx context.Context,
	phase, source, importInputKind, importFileExt string,
	duration time.Duration,
) {
	s.importerMetrics.RecordPhaseDuration(
		ctx,
		phase,
		normalizeImportSource(source),
		valueOrUnknown(importInputKind),
		valueOrUnknown(importFileExt),
		duration,
	)
}

func valueOrUnknown(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "unknown"
	}
	return v
}
