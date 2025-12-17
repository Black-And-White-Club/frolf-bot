package roundservice

import "time"

const (
	// Import behavior
	staleImportThreshold = 2 * time.Minute

	// Download limits
	downloadTimeout = 30 * time.Second
	maxRedirects    = 5
	maxFileSize     = 10 << 20 // 10MB

	// Error codes
	errCodeRoundNotFound     = "ROUND_NOT_FOUND"
	errCodeImportConflict    = "IMPORT_CONFLICT"
	errCodeDownloadError     = "DOWNLOAD_ERROR"
	errCodeDBError           = "DB_ERROR"
	errCodeUnsupported       = "UNSUPPORTED_FORMAT"
	errCodeFileTooLarge      = "FILE_TOO_LARGE"
	errCodeInvalidUDiscURL   = "INVALID_UDISC_URL"
	errCodeCtxCancelled      = "CTX_CANCELLED"
	errCodeImportApplyFailed = "IMPORT_APPLY_FAILED"
	errCodeParseError        = "PARSE_ERROR"
)
