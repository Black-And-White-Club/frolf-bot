package leaderboardservice

import (
	"errors"
	"fmt"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// Domain errors for the leaderboard service.
// These represent business logic failures that handlers should treat as
// normal outcomes (publish failure event, ack message) rather than retrying.
var (
	// ErrLeaderboardNotFound indicates a leaderboard does not exist.
	ErrLeaderboardNotFound = errors.New("leaderboard not found")

	// ErrInvalidGuildID indicates an empty or invalid guild ID was provided.
	ErrInvalidGuildID = errors.New("invalid guild ID")

	// ErrInvalidUserID indicates an empty or invalid user ID was provided.
	ErrInvalidUserID = errors.New("invalid user ID")

	// ErrTagNotAvailable indicates the requested tag is not available.
	ErrTagNotAvailable = errors.New("tag not available")

	// ErrCommandPipelineUnavailable indicates command-optimized orchestration is not configured.
	ErrCommandPipelineUnavailable = errors.New("command pipeline unavailable")
)

// TagSwapNeededError is returned when a requested tag is currently held by someone else.
// This triggers saga coordination for multi-party swaps.
type TagSwapNeededError struct {
	RequestorID  sharedtypes.DiscordID
	TargetUserID sharedtypes.DiscordID
	TargetTag    sharedtypes.TagNumber
	CurrentTag   sharedtypes.TagNumber
}

func (e *TagSwapNeededError) Error() string {
	return fmt.Sprintf("tag %d is held by %s; swap intent recorded for %s",
		e.TargetTag, e.TargetUserID, e.RequestorID)
}
