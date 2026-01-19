package leaderboarddb

import "errors"

// Sentinel errors for the repository layer.
// These represent infrastructure-level conditions callers may want
// to handle specially (not business-domain errors).
var (
	// ErrNotFound indicates the requested record does not exist.
	ErrNotFound = errors.New("not found")

	// ErrNoActiveLeaderboard indicates there is no active leaderboard for the guild.
	ErrNoActiveLeaderboard = errors.New("no active leaderboard found")

	// ErrUserTagNotFound indicates the requested user does not have a tag in the
	// active leaderboard.
	ErrUserTagNotFound = errors.New("user tag not found in active leaderboard")

	// ErrNoRowsAffected indicates an UPDATE/DELETE matched no rows.
	ErrNoRowsAffected = errors.New("no rows affected")
)
