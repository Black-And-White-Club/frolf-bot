// api/structs/round.go
package structs

import (
	"time"

	"github.com/Black-And-White-Club/tcr-bot/models"
)

// Round represents a single round in the tournament.
type Round struct {
	ID           int64              `json:"id"`
	Title        string             `json:"title"`
	Location     string             `json:"location"`
	EventType    *string            `json:"event_type"`
	Date         time.Time          `json:"date"`
	Time         string             `json:"time"`
	Finalized    bool               `json:"finalized"`
	CreatorID    string             `json:"creator_id"`
	State        models.RoundState  `json:"state"`
	Participants models.Participant `json:"participants"`
	Scores       map[string]int     `json:"scores"`
}
