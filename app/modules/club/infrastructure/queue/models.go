package clubqueue

import "github.com/google/uuid"

// OpenChallengeExpiryJob expires an open challenge after 48 hours.
type OpenChallengeExpiryJob struct {
	ChallengeID uuid.UUID `json:"challenge_id"`
}

func (OpenChallengeExpiryJob) Kind() string { return "club_challenge_open_expiry" }

// AcceptedChallengeExpiryJob expires an accepted unlinked challenge after 7 days.
type AcceptedChallengeExpiryJob struct {
	ChallengeID uuid.UUID `json:"challenge_id"`
}

func (AcceptedChallengeExpiryJob) Kind() string { return "club_challenge_accepted_expiry" }
