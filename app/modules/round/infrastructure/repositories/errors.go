package rounddb

import "errors"

// Sentinel errors for the repository layer.
// These are infrastructure-level errors that indicate database state, not business logic failures.
var (
	// ErrNotFound indicates the requested round does not exist in the database.
	ErrNotFound = errors.New("round not found")

	// ErrNoRowsAffected indicates an UPDATE or DELETE affected zero rows.
	ErrNoRowsAffected = errors.New("no rows affected")

	// ErrParticipantNotFound indicates the participant does not exist in the round.
	ErrParticipantNotFound = errors.New("participant not found")
)
