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

	// ErrAllScoresDNF indicates every participant in the round recorded a DNF,
	// leaving no active scores to process. This is a valid business outcome and
	// should not be treated as a data error.
	ErrAllScoresDNF = errors.New("all scores are DNF")
)
