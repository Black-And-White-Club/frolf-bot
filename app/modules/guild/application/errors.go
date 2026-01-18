package guildservice

import "errors"

// Domain errors for the guild service.
// These represent business logic failures that handlers should treat as
// normal outcomes (publish failure event, ack message) rather than retrying.
var (
	// ErrGuildConfigNotFound indicates a guild configuration does not exist yet.
	ErrGuildConfigNotFound = errors.New("guild config not found")

	// ErrGuildConfigConflict indicates attempted creation of a config that exists
	// with different settings. Use update instead.
	ErrGuildConfigConflict = errors.New("guild config already exists with different settings - use update instead")

	// ErrInvalidGuildID indicates an empty or invalid guild ID was provided.
	ErrInvalidGuildID = errors.New("invalid guild ID")

	// ErrNilConfig indicates a nil config was provided where one was required.
	ErrNilConfig = errors.New("config cannot be nil")
)
