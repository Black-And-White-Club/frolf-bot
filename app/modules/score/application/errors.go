package scoreservice

import "errors"

// Domain errors for the score service.
// These represent business logic failures that handlers should treat as
// normal outcomes (publish failure event, ack message) rather than retrying.
var (
	// ErrInvalidScore indicates the score value is outside valid bounds.
	ErrInvalidScore = errors.New("invalid score value")

	// ErrScoresAlreadyExist indicates scores already exist for a round and overwrite was not requested.
	ErrScoresAlreadyExist = errors.New("scores already exist for round")

	// ErrProcessingFailed indicates score processing failed during validation or storage preparation.
	ErrProcessingFailed = errors.New("score processing failed")
)
