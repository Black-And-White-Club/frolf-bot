package guildservice

import "errors"

// ErrGuildConfigNotFound indicates a guild configuration does not exist yet.
// Handlers should treat this as a normal domain failure (publish failure event, ack message) rather than retrying.
var ErrGuildConfigNotFound = errors.New("guild config not found")
