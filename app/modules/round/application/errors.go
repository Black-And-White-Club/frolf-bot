package roundservice

import (
	"errors"
	"fmt"
)

// Domain errors for the round service.
// These represent business logic failures that handlers should treat as
// normal outcomes (publish failure event, ack message) rather than retrying.
var (
	// ErrRoundNotFound indicates a round does not exist.
	ErrRoundNotFound = errors.New("round not found")

	// ErrInvalidRoundID indicates an empty or invalid round ID was provided.
	ErrInvalidRoundID = errors.New("invalid round ID")

	// ErrRoundAlreadyStarted indicates the round has already started.
	ErrRoundAlreadyStarted = errors.New("round has already started")

	// ErrRoundAlreadyFinalized indicates the round has already been finalized.
	ErrRoundAlreadyFinalized = errors.New("round has already been finalized")

	// ErrParticipantNotFound indicates the participant is not in the round.
	ErrParticipantNotFound = errors.New("participant not found in round")

	// ErrInvalidScore indicates an invalid score value was provided.
	ErrInvalidScore = errors.New("invalid score")

	// ErrUnauthorized indicates the user is not authorized for the operation.
	ErrUnauthorized = errors.New("unauthorized")
)

// ImportError is a structured error used internally by import helpers.
type ImportError struct {
	Code    string
	Message string
	Err     error
}

func (e *ImportError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *ImportError) Unwrap() error { return e.Err }
