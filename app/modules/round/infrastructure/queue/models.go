package roundqueue

import (
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
)

// RoundStartJob represents a scheduled round start event
// This job will publish a round.started event at the scheduled time
type RoundStartJob struct {
	RoundID   string                          `json:"round_id"`
	RoundData roundevents.RoundStartedPayload `json:"round_data"` // ✅ Correct payload type
}

// Kind returns the job type identifier for River
func (j RoundStartJob) Kind() string { return "round_start" }

// RoundReminderJob represents a scheduled round reminder event
// This job will publish a round.reminder event at the scheduled time
type RoundReminderJob struct {
	RoundID   string                             `json:"round_id"`
	RoundData roundevents.DiscordReminderPayload `json:"round_data"` // ✅ Correct payload type
}

// Kind returns the job type identifier for River
func (j RoundReminderJob) Kind() string { return "round_reminder" }

// JobInfo represents information about a scheduled job (for debugging/monitoring)
type JobInfo struct {
	ID          int64  `json:"id"`
	Kind        string `json:"kind"`
	RoundID     string `json:"round_id"`
	State       string `json:"state"`
	ScheduledAt string `json:"scheduled_at"`
	CreatedAt   string `json:"created_at"`
	Attempt     int    `json:"attempt"`
	MaxAttempts int    `json:"max_attempts"`
}
