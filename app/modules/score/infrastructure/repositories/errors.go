package scoredb

import "errors"

// Sentinel errors for the repository layer.
// These are infrastructure-level errors that indicate database state, not business logic failures.
var (
	// ErrNotFound indicates the requested score record does not exist in the database.
	// This is an infrastructure signal - the service layer decides if it's a domain failure.
	ErrNotFound = errors.New("score not found")

	// ErrNoRowsAffected indicates an UPDATE or DELETE affected zero rows.
	// Typically means the WHERE clause didn't match any active records.
	ErrNoRowsAffected = errors.New("no rows affected")
)
