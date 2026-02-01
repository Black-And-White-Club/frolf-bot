package userservice

import "errors"

// Domain errors for the user service.
// These represent business logic failures that handlers should treat as
// normal outcomes (publish failure event, ack message) rather than retrying.
var (
	// ErrUserAlreadyExists indicates a user already exists in the system.
	ErrUserAlreadyExists = errors.New("user already exists")

	// ErrInvalidDiscordID indicates an empty or invalid Discord ID was provided.
	ErrInvalidDiscordID = errors.New("Discord ID cannot be empty")

	// ErrNegativeTagNumber indicates a tag number cannot be negative.
	ErrNegativeTagNumber = errors.New("tag number cannot be negative")

	// ErrNilContext indicates a nil context was provided where one was required.
	ErrNilContext = errors.New("context cannot be nil")

	// ErrUserNotFound indicates the requested user does not exist.
	ErrUserNotFound = errors.New("user not found")

	// ErrInvalidRole indicates an invalid user role was provided.
	ErrInvalidRole = errors.New("invalid role")
)
