// models.go
package rounddb

import (
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// Round represents a single round in the tournament.
type Round struct {
	bun.BaseModel  `bun:"table:rounds,alias:r"`
	ID             sharedtypes.RoundID       `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	Title          roundtypes.Title          `bun:"title,notnull"`
	Description    roundtypes.Description    `bun:"description"`
	Location       roundtypes.Location       `bun:"location"`
	EventType      *roundtypes.EventType     `bun:"event_type"`
	StartTime      roundtypes.StartTime      `bun:"start_time,notnull"`
	Finalized      roundtypes.Finalized      `bun:"finalized,notnull"`
	CreatorID      sharedtypes.DiscordID     `bun:"created_by,notnull"`
	State          roundtypes.RoundState     `bun:"state,notnull"`
	Participants   []roundtypes.Participant  `bun:"participants,type:jsonb"`
	EventMessageID roundtypes.EventMessageID `bun:"event_message_id"`
	CreatedAt      time.Time                 `bun:",notnull,default:current_timestamp"`
	UpdatedAt      time.Time                 `bun:",notnull,default:current_timestamp"`
}

// Response represents the possible responses for a participant.
type Response string

// Define the possible response values as constants.
const (
	ResponseAccept    Response = "ACCEPT"
	ResponseTentative Response = "TENTATIVE"
	ResponseDecline   Response = "DECLINE"
)

// RoundState represents the state of a round.
type RoundState string

// Enum constants for RoundState
const (
	RoundStateUpcoming   RoundState = "UPCOMING"
	RoundStateInProgress RoundState = "IN_PROGRESS"
	RoundStateFinalized  RoundState = "FINALIZED"
	RoundStateDeleted    RoundState = "DELETED"
)

// Participant represents a user participating in a round.
type Participant struct {
	UserID    sharedtypes.DiscordID  `json:"user_id"`
	TagNumber *sharedtypes.TagNumber `json:"tag_number"`
	Response  Response               `json:"response"`
	Score     *sharedtypes.Score     `json:"score"`
}
